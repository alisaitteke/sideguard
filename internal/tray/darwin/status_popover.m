// status_popover.m — NSStatusItem + NSPopover panel with unified command review.
// See docs/plans/2026-07-03-1026-tray-command-review-unified/ (ucr-phase-1.0-darwin-tray.md).

#import "status_popover.h"
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#import <stdlib.h>

// Implemented in bindings.go via //export.
extern void goDecide(char *id, char *decision);
extern void goSetMode(int mode_index);
extern void goQuitTray(void);
extern void goInstallUpdate(void);
extern void goLoadMoreHistory(void);
extern void goAppearanceChanged(int dark);
extern void goOpenSettings(void);
extern void goSaveSettings(char *json);
extern void goAnalyseCommand(char *row_id, char *command, int use_event_id);

static char *copyCString(NSString *s) {
    if (s == nil) {
        return NULL;
    }
    return strdup([s UTF8String]);
}

static const CGFloat kPanelWidth = 360.0;
static const CGFloat kPanelHeightList = 480.0;
static const CGFloat kPanelHeightReview = 360.0;
static const CGFloat kReviewCommandMinHeight = 100.0;
static const CGFloat kPadding = 12.0;
static const CGFloat kRowHeight = 36.0;

static NSStatusItem *gStatusItem = nil;
static id gTrayDelegate = nil;
static NSPopover *gPopover = nil;
static darwin_ready_fn gOnReady = NULL;

static NSImageView *gHeaderLogoView = nil;
static NSTextField *gHeaderTitleLabel = nil;
static NSButton *gModeMenuButton = nil;
static NSMenu *gModeMenu = nil;
static NSStackView *gBodyStack = nil;
static NSScrollView *gBodyScroll = nil;
static NSView *gPanelRoot = nil;
static NSLayoutConstraint *gPanelHeightConstraint = nil;
static NSStackView *gListHeaderStack = nil;
static NSTextField *gFooterDaemonLabel = nil;
static NSTextField *gFooterPendingLabel = nil;
static NSStackView *gUpdateFooter = nil;
static NSTextField *gUpdateLabel = nil;
static NSButton *gInstallBtn = nil;

static BOOL gHistoryHasMore = NO;
static NSView *gBodyContainer = nil;
static NSButton *gAnalyseBtn = nil;
static NSTextField *gAnalyseStatusLabel = nil;
static NSTextField *gVerdictBadge = nil;
static NSTextField *gAnalyseSummaryLabel = nil;
static NSScrollView *gAnalyseExplanationScroll = nil;
static NSTextView *gAnalyseExplanationView = nil;
static BOOL gAnalyseInFlight = NO;
static NSView *gSettingsContainer = nil;
static NSScrollView *gSettingsScroll = nil;
static NSStackView *gSettingsRowsStack = nil;
static BOOL gShowingSettings = NO;
static NSString *gLastBodyFingerprint = nil;
static BOOL gHasSavedListScroll = NO;
static NSPoint gSavedListScrollOrigin = {0, 0};
static NSUInteger gLastPendingRowCount = 0;
static NSDictionary *gCachedTrayPayload = nil;

typedef NS_ENUM(NSInteger, CommandReviewMode) {
    CommandReviewModePending = 0,
    CommandReviewModeHistory = 1,
};

// Unified command review — pending auto-open and history row tap share this screen.
static NSView *gCommandReviewContainer = nil;
static NSButton *gReviewBackBtn = nil;
static NSTextField *gReviewCounter = nil;
static NSScrollView *gReviewCommandScroll = nil;
static NSTextView *gReviewCommandView = nil;
static NSStackView *gReviewDotsStack = nil;
static NSButton *gReviewPrevBtn = nil;
static NSButton *gReviewNextBtn = nil;
static NSButton *gReviewRunBtn = nil;
static NSButton *gReviewDeclineBtn = nil;
static NSStackView *gReviewCenterActions = nil;
static CommandReviewMode gReviewMode = CommandReviewModePending;
static NSArray *gReviewRows = nil;
static NSInteger gReviewIndex = 0;
static BOOL gReviewSkipMode = NO;
static BOOL gReviewDecided = NO;
static NSString *gReviewRowID = nil;
static BOOL gReviewUseEventID = NO;
static BOOL gShowingCommandReview = NO;
static char kRowDetailIndexKey;

static NSString *jsonString(id dict, NSString *key);
static BOOL jsonBool(id dict, NSString *key);
static NSInteger jsonInt(id dict, NSString *key);
static void rebuildBodyFromJSON(NSDictionary *payload, id target);
static NSString *bodyPayloadFingerprint(NSDictionary *payload);
static void restoreBodyScrollOrigin(NSClipView *clipView, NSPoint origin);
static void resetBodyScrollToTop(void);
static NSArray *pendingRowsFromPayload(NSDictionary *payload);
static NSArray *historyRowsFromPayload(NSDictionary *payload);
static void applyPanelLayoutForReviewMode(BOOL reviewMode);
static void clearListBodyStack(void);
static void enterCommandReviewPending(NSDictionary *payload, id target);
static void enterCommandReviewHistory(NSDictionary *payload, NSInteger index);
static void exitCommandReview(void);
static void syncCommandReviewNavButtons(void);
static void updateCommandReviewDots(void);
static void refreshCommandReviewSlide(NSDictionary *payload);
static void addRowsFromJSONArray(NSArray *rows, id target, NSMutableArray *outViews);
static NSView *makeHistoryRow(NSString *label, NSString *detail, NSString *rowID, NSInteger rowIndex, id target);
static NSView *makeLoadMoreRow(id target);
static void attachRowDetailTap(NSView *view, NSInteger rowIndex, id target);
static NSString *detailTextForRowID(NSDictionary *payload, NSString *rowID);
static BOOL isEffectiveAppearanceDark(NSView *view);
static void notifyAppearanceChanged(void);
static void applySemanticTextColors(void);
static void hideSettingsView(void);
static void rebuildSettingsRowsFromJSON(NSDictionary *payload, id target);
static NSView *makeSettingsProviderRow(NSDictionary *row, NSArray *drivers, id target, NSUInteger rowIndex);
static NSDictionary *collectSettingsSavePayload(void);
static void clearAnalyseUI(void);
static void styleColoredActionButton(NSButton *btn, NSString *title, NSColor *accentColor);
static NSString *verdictDisplayTitle(NSString *verdict);
static NSColor *verdictBadgeColor(NSString *verdict);
static NSDictionary *reviewRowAtIndex(NSArray *rows, NSInteger index);

@interface SideGuardPanelRootView : NSView
@end

@implementation SideGuardPanelRootView

- (void)viewDidChangeEffectiveAppearance {
    [super viewDidChangeEffectiveAppearance];
    notifyAppearanceChanged();
}

@end

@interface SideGuardTrayDelegate : NSObject <NSApplicationDelegate, NSPopoverDelegate>
@end

@implementation SideGuardTrayDelegate

- (void)dealloc {
    [[NSNotificationCenter defaultCenter] removeObserver:self];
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    (void)notification;

    gStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    NSStatusBarButton *button = gStatusItem.button;
    button.target = self;
    button.action = @selector(togglePopover:);

    gPopover = [[NSPopover alloc] init];
    gPopover.behavior = NSPopoverBehaviorTransient;
    gPopover.delegate = self;

    [self buildPanelShell];

    NSViewController *vc = [[NSViewController alloc] init];
    vc.view = gPanelRoot;
    gPopover.contentViewController = vc;

    if (gOnReady != NULL) {
        gOnReady();
    }

    notifyAppearanceChanged();
}

- (void)buildPanelShell {
    gPanelRoot = [[SideGuardPanelRootView alloc] initWithFrame:NSMakeRect(0, 0, kPanelWidth, kPanelHeightList)];

    gPanelHeightConstraint = [gPanelRoot.heightAnchor constraintEqualToConstant:kPanelHeightList];
    [NSLayoutConstraint activateConstraints:@[
        [gPanelRoot.widthAnchor constraintEqualToConstant:kPanelWidth],
        gPanelHeightConstraint,
    ]];

    NSStackView *rootStack = [[NSStackView alloc] init];
    rootStack.translatesAutoresizingMaskIntoConstraints = NO;
    rootStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    rootStack.alignment = NSLayoutAttributeLeading;
    rootStack.spacing = 8.0;
    rootStack.edgeInsets = NSEdgeInsetsMake(kPadding, kPadding, kPadding, kPadding);
    [gPanelRoot addSubview:rootStack];

    [NSLayoutConstraint activateConstraints:@[
        [rootStack.topAnchor constraintEqualToAnchor:gPanelRoot.topAnchor],
        [rootStack.bottomAnchor constraintEqualToAnchor:gPanelRoot.bottomAnchor],
        [rootStack.leadingAnchor constraintEqualToAnchor:gPanelRoot.leadingAnchor],
        [rootStack.trailingAnchor constraintEqualToAnchor:gPanelRoot.trailingAnchor],
    ]];

    // Header: title (leading) + hamburger menu (trailing) with mode + Quit.
    NSStackView *headerStack = [[NSStackView alloc] init];
    headerStack.translatesAutoresizingMaskIntoConstraints = NO;
    headerStack.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    headerStack.alignment = NSLayoutAttributeCenterY;
    headerStack.spacing = 8.0;

    gModeMenu = [[NSMenu alloc] init];
    NSArray *modeTitles = @[@"Ask", @"Auto", @"Auto-allow", @"Auto-deny"];
    for (NSUInteger i = 0; i < modeTitles.count; i++) {
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:modeTitles[i]
                                                      action:@selector(modeMenuItemSelected:)
                                               keyEquivalent:@""];
        item.target = self;
        item.tag = (NSInteger)i;
        [gModeMenu addItem:item];
    }
    [gModeMenu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *settingsItem = [[NSMenuItem alloc] initWithTitle:@"Settings…"
                                                          action:@selector(settingsMenuSelected:)
                                                   keyEquivalent:@""];
    settingsItem.target = self;
    [gModeMenu addItem:settingsItem];
    [gModeMenu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
                                                      action:@selector(quitClicked:)
                                               keyEquivalent:@""];
    quitItem.target = self;
    [gModeMenu addItem:quitItem];

    gHeaderLogoView = [[NSImageView alloc] init];
    gHeaderLogoView.translatesAutoresizingMaskIntoConstraints = NO;
    gHeaderLogoView.imageScaling = NSImageScaleProportionallyUpOrDown;
    [gHeaderLogoView setContentHuggingPriority:NSLayoutPriorityRequired
                                  forOrientation:NSLayoutConstraintOrientationHorizontal];
    [headerStack addArrangedSubview:gHeaderLogoView];
    [NSLayoutConstraint activateConstraints:@[
        [gHeaderLogoView.widthAnchor constraintEqualToConstant:24.0],
        [gHeaderLogoView.heightAnchor constraintEqualToConstant:24.0],
    ]];

    gHeaderTitleLabel = [NSTextField labelWithString:@"SideGuard"];
    gHeaderTitleLabel.translatesAutoresizingMaskIntoConstraints = NO;
    gHeaderTitleLabel.font = [NSFont systemFontOfSize:15.0 weight:NSFontWeightSemibold];
    gHeaderTitleLabel.textColor = [NSColor labelColor];
    [gHeaderTitleLabel setContentHuggingPriority:NSLayoutPriorityDefaultHigh
                                   forOrientation:NSLayoutConstraintOrientationHorizontal];
    [headerStack addArrangedSubview:gHeaderTitleLabel];

    NSView *headerSpacer = [[NSView alloc] init];
    [headerSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow
                             forOrientation:NSLayoutConstraintOrientationHorizontal];
    [headerStack addArrangedSubview:headerSpacer];

    gModeMenuButton = [NSButton buttonWithTitle:@"≡" target:self action:@selector(showModeMenu:)];
    gModeMenuButton.bezelStyle = NSBezelStyleRounded;
    gModeMenuButton.font = [NSFont boldSystemFontOfSize:16.0];
    [gModeMenuButton setContentHuggingPriority:NSLayoutPriorityRequired
                                 forOrientation:NSLayoutConstraintOrientationHorizontal];
    [headerStack addArrangedSubview:gModeMenuButton];

    [rootStack addArrangedSubview:headerStack];
    [headerStack.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;

    // Body area: list (scrollable rows) or command detail view.
    gBodyContainer = [[NSView alloc] init];
    gBodyContainer.translatesAutoresizingMaskIntoConstraints = NO;

    gBodyStack = [[NSStackView alloc] init];
    gBodyStack.translatesAutoresizingMaskIntoConstraints = NO;
    gBodyStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    gBodyStack.alignment = NSLayoutAttributeLeading;
    gBodyStack.spacing = 6.0;

    gBodyScroll = [[NSScrollView alloc] init];
    gBodyScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gBodyScroll.hasVerticalScroller = YES;
    gBodyScroll.autohidesScrollers = YES;
    gBodyScroll.drawsBackground = NO;
    gBodyScroll.borderType = NSNoBorder;
    gBodyScroll.documentView = gBodyStack;
    NSView *clipView = gBodyScroll.contentView;
    // Never observe clip bounds — programmatic scroll restore must not auto-load history.
    [clipView setPostsBoundsChangedNotifications:NO];
    [gBodyContainer addSubview:gBodyScroll];

    gListHeaderStack = [[NSStackView alloc] init];
    gListHeaderStack.translatesAutoresizingMaskIntoConstraints = NO;
    gListHeaderStack.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    gListHeaderStack.alignment = NSLayoutAttributeCenterY;
    gListHeaderStack.spacing = 8.0;
    [gBodyContainer addSubview:gListHeaderStack];

    NSTextField *historyHeaderLabel = [NSTextField labelWithString:@"History"];
    historyHeaderLabel.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize] weight:NSFontWeightSemibold];
    historyHeaderLabel.textColor = [NSColor tertiaryLabelColor];
    [gListHeaderStack addArrangedSubview:historyHeaderLabel];

    NSView *listHeaderSpacer = [[NSView alloc] init];
    [listHeaderSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow
                               forOrientation:NSLayoutConstraintOrientationHorizontal];
    [gListHeaderStack addArrangedSubview:listHeaderSpacer];

    gFooterDaemonLabel = [self makeStatusLabel:@"● Daemon: …"];
    gFooterPendingLabel = [self makeStatusLabel:@"● pending …"];
    [gFooterPendingLabel setContentCompressionResistancePriority:NSLayoutPriorityRequired
                                                forOrientation:NSLayoutConstraintOrientationHorizontal];
    [gListHeaderStack addArrangedSubview:gFooterDaemonLabel];
    [gListHeaderStack addArrangedSubview:gFooterPendingLabel];

    [NSLayoutConstraint activateConstraints:@[
        [gListHeaderStack.topAnchor constraintEqualToAnchor:gBodyContainer.topAnchor],
        [gListHeaderStack.leadingAnchor constraintEqualToAnchor:gBodyContainer.leadingAnchor],
        [gListHeaderStack.trailingAnchor constraintEqualToAnchor:gBodyContainer.trailingAnchor],
        [gListHeaderStack.heightAnchor constraintEqualToConstant:22.0],

        [gBodyScroll.topAnchor constraintEqualToAnchor:gListHeaderStack.bottomAnchor constant:4.0],
        [gBodyScroll.bottomAnchor constraintEqualToAnchor:gBodyContainer.bottomAnchor],
        [gBodyScroll.leadingAnchor constraintEqualToAnchor:gBodyContainer.leadingAnchor],
        [gBodyScroll.trailingAnchor constraintEqualToAnchor:gBodyContainer.trailingAnchor],
    ]];
  // Pin width/top only — document height comes from stack content so the list scrolls.
    [NSLayoutConstraint activateConstraints:@[
        [gBodyStack.leadingAnchor constraintEqualToAnchor:clipView.leadingAnchor],
        [gBodyStack.trailingAnchor constraintEqualToAnchor:clipView.trailingAnchor],
        [gBodyStack.topAnchor constraintEqualToAnchor:clipView.topAnchor],
        [gBodyStack.widthAnchor constraintEqualToAnchor:clipView.widthAnchor],
    ]];

    gSettingsContainer = [[NSView alloc] init];
    gSettingsContainer.translatesAutoresizingMaskIntoConstraints = NO;
    gSettingsContainer.hidden = YES;
    [gBodyContainer addSubview:gSettingsContainer];
    [NSLayoutConstraint activateConstraints:@[
        [gSettingsContainer.topAnchor constraintEqualToAnchor:gBodyContainer.topAnchor],
        [gSettingsContainer.bottomAnchor constraintEqualToAnchor:gBodyContainer.bottomAnchor],
        [gSettingsContainer.leadingAnchor constraintEqualToAnchor:gBodyContainer.leadingAnchor],
        [gSettingsContainer.trailingAnchor constraintEqualToAnchor:gBodyContainer.trailingAnchor],
    ]];

    NSStackView *settingsOuterStack = [[NSStackView alloc] init];
    settingsOuterStack.translatesAutoresizingMaskIntoConstraints = NO;
    settingsOuterStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    settingsOuterStack.alignment = NSLayoutAttributeLeading;
    settingsOuterStack.spacing = 8.0;
    [gSettingsContainer addSubview:settingsOuterStack];
    [NSLayoutConstraint activateConstraints:@[
        [settingsOuterStack.topAnchor constraintEqualToAnchor:gSettingsContainer.topAnchor],
        [settingsOuterStack.bottomAnchor constraintEqualToAnchor:gSettingsContainer.bottomAnchor],
        [settingsOuterStack.leadingAnchor constraintEqualToAnchor:gSettingsContainer.leadingAnchor],
        [settingsOuterStack.trailingAnchor constraintEqualToAnchor:gSettingsContainer.trailingAnchor],
    ]];

    NSButton *settingsBackBtn = [NSButton buttonWithTitle:@"← Back" target:self action:@selector(settingsBackClicked:)];
    settingsBackBtn.bezelStyle = NSBezelStyleRounded;
    settingsBackBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [settingsOuterStack addArrangedSubview:settingsBackBtn];

    NSTextField *settingsTitle = [NSTextField labelWithString:@"LLM Providers"];
    settingsTitle.font = [NSFont systemFontOfSize:14.0 weight:NSFontWeightSemibold];
    settingsTitle.textColor = [NSColor labelColor];
    [settingsOuterStack addArrangedSubview:settingsTitle];

    gSettingsScroll = [[NSScrollView alloc] init];
    gSettingsScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gSettingsScroll.hasVerticalScroller = YES;
    gSettingsScroll.autohidesScrollers = YES;
    gSettingsScroll.drawsBackground = NO;
    gSettingsScroll.borderType = NSNoBorder;

    gSettingsRowsStack = [[NSStackView alloc] init];
    gSettingsRowsStack.translatesAutoresizingMaskIntoConstraints = NO;
    gSettingsRowsStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    gSettingsRowsStack.alignment = NSLayoutAttributeLeading;
    gSettingsRowsStack.spacing = 10.0;
    gSettingsScroll.documentView = gSettingsRowsStack;
    NSView *settingsClip = gSettingsScroll.contentView;
    [NSLayoutConstraint activateConstraints:@[
        [gSettingsRowsStack.leadingAnchor constraintEqualToAnchor:settingsClip.leadingAnchor],
        [gSettingsRowsStack.trailingAnchor constraintEqualToAnchor:settingsClip.trailingAnchor],
        [gSettingsRowsStack.topAnchor constraintEqualToAnchor:settingsClip.topAnchor],
        [gSettingsRowsStack.widthAnchor constraintEqualToAnchor:settingsClip.widthAnchor],
    ]];
    [settingsOuterStack addArrangedSubview:gSettingsScroll];
    [gSettingsScroll setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationVertical];
    [gSettingsScroll setContentCompressionResistancePriority:NSLayoutPriorityDefaultLow
                                            forOrientation:NSLayoutConstraintOrientationVertical];

    NSStackView *settingsActions = [[NSStackView alloc] init];
    settingsActions.translatesAutoresizingMaskIntoConstraints = NO;
    settingsActions.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    settingsActions.alignment = NSLayoutAttributeCenterY;
    settingsActions.spacing = 8.0;

    NSButton *addProviderBtn = [NSButton buttonWithTitle:@"+ Add provider" target:self action:@selector(settingsAddProvider:)];
    addProviderBtn.bezelStyle = NSBezelStyleRounded;
    [settingsActions addArrangedSubview:addProviderBtn];

    NSView *settingsActionSpacer = [[NSView alloc] init];
    [settingsActionSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow
                                     forOrientation:NSLayoutConstraintOrientationHorizontal];
    [settingsActions addArrangedSubview:settingsActionSpacer];

    NSButton *saveSettingsBtn = [NSButton buttonWithTitle:@"Save" target:self action:@selector(settingsSave:)];
    saveSettingsBtn.bezelStyle = NSBezelStyleRounded;
    [settingsActions addArrangedSubview:saveSettingsBtn];
    [settingsOuterStack addArrangedSubview:settingsActions];
    [settingsActions.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;

    [rootStack addArrangedSubview:gBodyContainer];
    [gBodyContainer setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationVertical];
    [gBodyContainer setContentCompressionResistancePriority:NSLayoutPriorityDefaultLow
                                             forOrientation:NSLayoutConstraintOrientationVertical];

    // Unified command review — overlays list body (same slot, no extra root-stack row).
    gCommandReviewContainer = [[NSView alloc] init];
    gCommandReviewContainer.translatesAutoresizingMaskIntoConstraints = NO;
    gCommandReviewContainer.hidden = YES;
    [gBodyContainer addSubview:gCommandReviewContainer];

    gReviewBackBtn = [NSButton buttonWithTitle:@"← History" target:self action:@selector(reviewBackClicked:)];
    gReviewBackBtn.bezelStyle = NSBezelStyleRounded;
    gReviewBackBtn.translatesAutoresizingMaskIntoConstraints = NO;
    gReviewBackBtn.hidden = YES;
    [gCommandReviewContainer addSubview:gReviewBackBtn];

    gReviewCounter = [NSTextField labelWithString:@"1 of 1"];
    gReviewCounter.translatesAutoresizingMaskIntoConstraints = NO;
    gReviewCounter.font = [NSFont monospacedDigitSystemFontOfSize:[NSFont smallSystemFontSize]
                                                            weight:NSFontWeightMedium];
    gReviewCounter.textColor = [NSColor secondaryLabelColor];
    gReviewCounter.alignment = NSTextAlignmentRight;
    [gCommandReviewContainer addSubview:gReviewCounter];

    gReviewCommandScroll = [[NSScrollView alloc] init];
    gReviewCommandScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gReviewCommandScroll.hasVerticalScroller = YES;
    gReviewCommandScroll.autohidesScrollers = YES;
    gReviewCommandScroll.borderType = NSNoBorder;
    gReviewCommandScroll.wantsLayer = YES;
    gReviewCommandScroll.layer.cornerRadius = 8.0;
    gReviewCommandScroll.layer.borderWidth = 1.0;
    gReviewCommandScroll.layer.borderColor = [[NSColor separatorColor] colorWithAlphaComponent:0.6].CGColor;
    gReviewCommandScroll.drawsBackground = YES;
    gReviewCommandScroll.backgroundColor = [[NSColor textBackgroundColor] colorWithAlphaComponent:0.55];

    gReviewCommandView = [[NSTextView alloc] init];
    gReviewCommandView.editable = NO;
    gReviewCommandView.selectable = YES;
    gReviewCommandView.drawsBackground = NO;
    gReviewCommandView.font = [NSFont monospacedSystemFontOfSize:12.0 weight:NSFontWeightRegular];
    gReviewCommandView.textColor = [NSColor labelColor];
    gReviewCommandView.textContainerInset = NSMakeSize(10.0, 10.0);
    gReviewCommandView.textContainer.widthTracksTextView = YES;
    gReviewCommandView.textContainer.containerSize = NSMakeSize(0, CGFLOAT_MAX);
    gReviewCommandView.verticallyResizable = YES;
    gReviewCommandView.horizontallyResizable = NO;
    gReviewCommandScroll.documentView = gReviewCommandView;
    [gCommandReviewContainer addSubview:gReviewCommandScroll];

    gAnalyseBtn = [NSButton buttonWithTitle:@"Analyse" target:self action:@selector(analyseClicked:)];
    gAnalyseBtn.bezelStyle = NSBezelStyleRounded;
    gAnalyseBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [gCommandReviewContainer addSubview:gAnalyseBtn];

    gAnalyseStatusLabel = [NSTextField labelWithString:@""];
    gAnalyseStatusLabel.translatesAutoresizingMaskIntoConstraints = NO;
    gAnalyseStatusLabel.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    gAnalyseStatusLabel.textColor = [NSColor secondaryLabelColor];
    gAnalyseStatusLabel.hidden = YES;
    [gCommandReviewContainer addSubview:gAnalyseStatusLabel];

    gVerdictBadge = [NSTextField labelWithString:@""];
    gVerdictBadge.translatesAutoresizingMaskIntoConstraints = NO;
    gVerdictBadge.font = [NSFont systemFontOfSize:[NSFont systemFontSize] weight:NSFontWeightSemibold];
    gVerdictBadge.hidden = YES;
    [gCommandReviewContainer addSubview:gVerdictBadge];

    gAnalyseSummaryLabel = [NSTextField labelWithString:@""];
    gAnalyseSummaryLabel.translatesAutoresizingMaskIntoConstraints = NO;
    gAnalyseSummaryLabel.font = [NSFont systemFontOfSize:[NSFont systemFontSize] weight:NSFontWeightMedium];
    gAnalyseSummaryLabel.textColor = [NSColor labelColor];
    gAnalyseSummaryLabel.hidden = YES;
    [gCommandReviewContainer addSubview:gAnalyseSummaryLabel];

    gAnalyseExplanationScroll = [[NSScrollView alloc] init];
    gAnalyseExplanationScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gAnalyseExplanationScroll.hasVerticalScroller = YES;
    gAnalyseExplanationScroll.autohidesScrollers = YES;
    gAnalyseExplanationScroll.drawsBackground = NO;
    gAnalyseExplanationScroll.borderType = NSNoBorder;
    gAnalyseExplanationScroll.hidden = YES;

    gAnalyseExplanationView = [[NSTextView alloc] init];
    gAnalyseExplanationView.editable = NO;
    gAnalyseExplanationView.selectable = YES;
    gAnalyseExplanationView.drawsBackground = NO;
    gAnalyseExplanationView.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    gAnalyseExplanationView.textColor = [NSColor secondaryLabelColor];
    gAnalyseExplanationView.textContainerInset = NSMakeSize(2.0, 2.0);
    gAnalyseExplanationView.textContainer.widthTracksTextView = YES;
    gAnalyseExplanationView.textContainer.containerSize = NSMakeSize(0, CGFLOAT_MAX);
    gAnalyseExplanationView.verticallyResizable = YES;
    gAnalyseExplanationView.horizontallyResizable = NO;
    gAnalyseExplanationScroll.documentView = gAnalyseExplanationView;
    [gCommandReviewContainer addSubview:gAnalyseExplanationScroll];

    gReviewDotsStack = [[NSStackView alloc] init];
    gReviewDotsStack.translatesAutoresizingMaskIntoConstraints = NO;
    gReviewDotsStack.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    gReviewDotsStack.alignment = NSLayoutAttributeCenterY;
    gReviewDotsStack.spacing = 5.0;
    [gCommandReviewContainer addSubview:gReviewDotsStack];

    NSStackView *reviewNavBar = [[NSStackView alloc] init];
    reviewNavBar.translatesAutoresizingMaskIntoConstraints = NO;
    reviewNavBar.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    reviewNavBar.alignment = NSLayoutAttributeCenterY;
    reviewNavBar.distribution = NSStackViewDistributionFillEqually;
    reviewNavBar.spacing = 8.0;
    [gCommandReviewContainer addSubview:reviewNavBar];

    gReviewPrevBtn = [NSButton buttonWithTitle:@"Previous" target:self action:@selector(reviewPrevClicked:)];
    gReviewPrevBtn.bezelStyle = NSBezelStyleRounded;
    gReviewPrevBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [reviewNavBar addArrangedSubview:gReviewPrevBtn];

    gReviewCenterActions = [[NSStackView alloc] init];
    gReviewCenterActions.orientation = NSUserInterfaceLayoutOrientationVertical;
    gReviewCenterActions.alignment = NSLayoutAttributeCenterX;
    gReviewCenterActions.spacing = 6.0;
    [reviewNavBar addArrangedSubview:gReviewCenterActions];

    gReviewRunBtn = [NSButton buttonWithTitle:@"Run" target:self action:@selector(reviewRunClicked:)];
    gReviewRunBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(gReviewRunBtn, @"Run", [NSColor systemGreenColor]);
    [gReviewCenterActions addArrangedSubview:gReviewRunBtn];
    [gReviewRunBtn.widthAnchor constraintGreaterThanOrEqualToConstant:92.0].active = YES;

    gReviewDeclineBtn = [NSButton buttonWithTitle:@"Decline" target:self action:@selector(reviewDeclineClicked:)];
    gReviewDeclineBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(gReviewDeclineBtn, @"Decline", [NSColor systemRedColor]);
    [gReviewCenterActions addArrangedSubview:gReviewDeclineBtn];
    [gReviewDeclineBtn.widthAnchor constraintGreaterThanOrEqualToConstant:92.0].active = YES;

    gReviewNextBtn = [NSButton buttonWithTitle:@"Next" target:self action:@selector(reviewNextClicked:)];
    gReviewNextBtn.bezelStyle = NSBezelStyleRounded;
    gReviewNextBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [reviewNavBar addArrangedSubview:gReviewNextBtn];

    [NSLayoutConstraint activateConstraints:@[
        [gReviewBackBtn.topAnchor constraintEqualToAnchor:gCommandReviewContainer.topAnchor],
        [gReviewBackBtn.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],

        [gReviewCounter.topAnchor constraintEqualToAnchor:gReviewBackBtn.bottomAnchor constant:4.0],
        [gReviewCounter.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],
        [gReviewCounter.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],

        [gReviewCommandScroll.topAnchor constraintEqualToAnchor:gReviewCounter.bottomAnchor constant:6.0],
        [gReviewCommandScroll.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [gReviewCommandScroll.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],
        [gReviewCommandScroll.heightAnchor constraintGreaterThanOrEqualToConstant:kReviewCommandMinHeight],

        [gAnalyseBtn.topAnchor constraintEqualToAnchor:gReviewCommandScroll.bottomAnchor constant:8.0],
        [gAnalyseBtn.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],

        [gAnalyseStatusLabel.topAnchor constraintEqualToAnchor:gAnalyseBtn.bottomAnchor constant:4.0],
        [gAnalyseStatusLabel.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [gAnalyseStatusLabel.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],

        [gVerdictBadge.topAnchor constraintEqualToAnchor:gAnalyseStatusLabel.bottomAnchor constant:2.0],
        [gVerdictBadge.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [gVerdictBadge.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],

        [gAnalyseSummaryLabel.topAnchor constraintEqualToAnchor:gVerdictBadge.bottomAnchor constant:2.0],
        [gAnalyseSummaryLabel.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [gAnalyseSummaryLabel.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],

        [gAnalyseExplanationScroll.topAnchor constraintEqualToAnchor:gAnalyseSummaryLabel.bottomAnchor constant:2.0],
        [gAnalyseExplanationScroll.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [gAnalyseExplanationScroll.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],
        [gAnalyseExplanationScroll.heightAnchor constraintLessThanOrEqualToConstant:72.0],

        [gReviewDotsStack.topAnchor constraintEqualToAnchor:gAnalyseExplanationScroll.bottomAnchor constant:6.0],
        [gReviewDotsStack.centerXAnchor constraintEqualToAnchor:gCommandReviewContainer.centerXAnchor],

        [reviewNavBar.topAnchor constraintEqualToAnchor:gReviewDotsStack.bottomAnchor constant:6.0],
        [reviewNavBar.bottomAnchor constraintEqualToAnchor:gCommandReviewContainer.bottomAnchor],
        [reviewNavBar.leadingAnchor constraintEqualToAnchor:gCommandReviewContainer.leadingAnchor],
        [reviewNavBar.trailingAnchor constraintEqualToAnchor:gCommandReviewContainer.trailingAnchor],
        [reviewNavBar.heightAnchor constraintEqualToConstant:64.0],

        [gCommandReviewContainer.topAnchor constraintEqualToAnchor:gBodyContainer.topAnchor],
        [gCommandReviewContainer.bottomAnchor constraintEqualToAnchor:gBodyContainer.bottomAnchor],
        [gCommandReviewContainer.leadingAnchor constraintEqualToAnchor:gBodyContainer.leadingAnchor],
        [gCommandReviewContainer.trailingAnchor constraintEqualToAnchor:gBodyContainer.trailingAnchor],
    ]];

    gUpdateFooter = [[NSStackView alloc] init];
    gUpdateFooter.translatesAutoresizingMaskIntoConstraints = NO;
    gUpdateFooter.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    gUpdateFooter.alignment = NSLayoutAttributeCenterY;
    gUpdateFooter.spacing = 8.0;
    gUpdateFooter.hidden = YES;

    gUpdateLabel = [self makeStatusLabel:@""];
    [gUpdateFooter addArrangedSubview:gUpdateLabel];

    gInstallBtn = [NSButton buttonWithTitle:@"Install" target:self action:@selector(installClicked:)];
    gInstallBtn.bezelStyle = NSBezelStyleRounded;
    [gUpdateFooter addArrangedSubview:gInstallBtn];
    [gInstallBtn setContentHuggingPriority:NSLayoutPriorityRequired forOrientation:NSLayoutConstraintOrientationHorizontal];

    [rootStack addArrangedSubview:gUpdateFooter];
    [gUpdateFooter.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;
}

- (NSTextField *)makeStatusLabel:(NSString *)text {
    NSTextField *field = [NSTextField labelWithString:text];
    field.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    field.textColor = [NSColor secondaryLabelColor];
    field.lineBreakMode = NSLineBreakByTruncatingTail;
    return field;
}

- (void)togglePopover:(id)sender {
    (void)sender;
    if (gPopover == nil || gStatusItem == nil) {
        return;
    }
    if (gPopover.isShown) {
        [gPopover performClose:nil];
        return;
    }
    NSStatusBarButton *button = gStatusItem.button;
    [gPopover showRelativeToRect:button.bounds
                          ofView:button
                   preferredEdge:NSMinYEdge];
}

- (void)showModeMenu:(id)sender {
    (void)sender;
    if (gModeMenu == nil || gModeMenuButton == nil) {
        return;
    }
    NSRect frame = gModeMenuButton.bounds;
    NSPoint location = NSMakePoint(NSMinX(frame), NSMinY(frame));
    [gModeMenu popUpMenuPositioningItem:nil atLocation:location inView:gModeMenuButton];
}

- (void)modeMenuItemSelected:(NSMenuItem *)sender {
    goSetMode((int)sender.tag);
}

- (void)loadMoreClicked:(NSClickGestureRecognizer *)sender {
    if (sender != nil && sender.state != NSGestureRecognizerStateEnded) {
        return;
    }
    if (!gHistoryHasMore) {
        return;
    }
    goLoadMoreHistory();
}

- (void)allowClicked:(NSButton *)sender {
    NSString *approvalID = sender.identifier;
    if (approvalID.length == 0) {
        return;
    }
    char *idCopy = copyCString(approvalID);
    if (idCopy == NULL) {
        return;
    }
    goDecide(idCopy, "allow");
    free(idCopy);
}

- (void)denyClicked:(NSButton *)sender {
    NSString *approvalID = sender.identifier;
    if (approvalID.length == 0) {
        return;
    }
    char *idCopy = copyCString(approvalID);
    if (idCopy == NULL) {
        return;
    }
    goDecide(idCopy, "deny");
    free(idCopy);
}

- (void)reviewRunClicked:(NSButton *)sender {
    gReviewDecided = YES;
    syncCommandReviewNavButtons();
    [self allowClicked:sender];
}

- (void)reviewDeclineClicked:(NSButton *)sender {
    gReviewDecided = YES;
    syncCommandReviewNavButtons();
    [self denyClicked:sender];
}

- (void)reviewPrevClicked:(id)sender {
    (void)sender;
    if (gReviewRows.count == 0) {
        return;
    }
    if (gReviewMode == CommandReviewModePending) {
        if (gReviewIndex <= 0 || gReviewDecided) {
            return;
        }
        gReviewIndex -= 1;
        gReviewSkipMode = NO;
        gReviewDecided = NO;
    } else {
        if (gReviewIndex <= 0) {
            return;
        }
        gReviewIndex -= 1;
    }
    refreshCommandReviewSlide(gCachedTrayPayload);
}

- (void)reviewNextClicked:(id)sender {
    (void)sender;
    if (gReviewRows.count == 0) {
        return;
    }
    if (gReviewMode == CommandReviewModePending) {
        if (gReviewSkipMode || gReviewDecided) {
            return;
        }
        if ((NSUInteger)gReviewIndex >= gReviewRows.count - 1) {
            return;
        }
        gReviewIndex += 1;
        gReviewSkipMode = YES;
        gReviewDecided = NO;
    } else {
        if ((NSUInteger)gReviewIndex >= gReviewRows.count - 1) {
            return;
        }
        gReviewIndex += 1;
    }
    refreshCommandReviewSlide(gCachedTrayPayload);
}

- (void)reviewBackClicked:(id)sender {
    (void)sender;
    exitCommandReview();
}

- (void)quitClicked:(id)sender {
    (void)sender;
    NSAlert *alert = [[NSAlert alloc] init];
    alert.alertStyle = NSAlertStyleWarning;
    alert.messageText = @"Quit SideGuard tray?";
    alert.informativeText = @"Approvals will remain queued; reopen tray from the app.";
    [alert addButtonWithTitle:@"Cancel"];
    [alert addButtonWithTitle:@"Quit"];
    NSModalResponse response = [alert runModal];
    if (response == NSAlertSecondButtonReturn) {
        goQuitTray();
    }
}

- (void)installClicked:(id)sender {
    (void)sender;
    goInstallUpdate();
}

- (void)analyseClicked:(id)sender {
    (void)sender;
    if (gAnalyseInFlight) {
        return;
    }
    gAnalyseInFlight = YES;
    if (gAnalyseBtn != nil) {
        gAnalyseBtn.enabled = NO;
    }
    if (gVerdictBadge != nil) {
        gVerdictBadge.hidden = YES;
    }
    if (gAnalyseSummaryLabel != nil) {
        gAnalyseSummaryLabel.hidden = YES;
        gAnalyseSummaryLabel.stringValue = @"";
    }
    if (gAnalyseExplanationScroll != nil) {
        gAnalyseExplanationScroll.hidden = YES;
    }
    if (gAnalyseExplanationView != nil) {
        gAnalyseExplanationView.string = @"";
    }
    if (gAnalyseStatusLabel != nil) {
        gAnalyseStatusLabel.hidden = NO;
        gAnalyseStatusLabel.stringValue = @"Analysing…";
        gAnalyseStatusLabel.textColor = [NSColor secondaryLabelColor];
    }

    NSString *rowID = gReviewRowID ?: @"";
    NSString *command = gReviewCommandView.string ?: @"";
    char *rowCopy = copyCString(rowID);
    char *cmdCopy = copyCString(command);
    if (rowCopy == NULL || cmdCopy == NULL) {
        gAnalyseInFlight = NO;
        if (gAnalyseBtn != nil) {
            gAnalyseBtn.enabled = YES;
        }
        free(rowCopy);
        free(cmdCopy);
        return;
    }
    goAnalyseCommand(rowCopy, cmdCopy, gReviewUseEventID ? 1 : 0);
    free(rowCopy);
    free(cmdCopy);
}

- (void)updateAnalyseResult:(NSString *)jsonStr {
    gAnalyseInFlight = NO;
    if (gAnalyseBtn != nil) {
        gAnalyseBtn.enabled = YES;
    }
    if (jsonStr.length == 0) {
        if (gAnalyseStatusLabel != nil) {
            gAnalyseStatusLabel.hidden = NO;
            gAnalyseStatusLabel.stringValue = @"Analysis failed.";
            gAnalyseStatusLabel.textColor = [NSColor systemRedColor];
        }
        return;
    }

    NSData *data = [jsonStr dataUsingEncoding:NSUTF8StringEncoding];
    NSError *err = nil;
    id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:&err];
    if (err != nil || ![parsed isKindOfClass:[NSDictionary class]]) {
        if (gAnalyseStatusLabel != nil) {
            gAnalyseStatusLabel.hidden = NO;
            gAnalyseStatusLabel.stringValue = @"Could not read analysis result.";
            gAnalyseStatusLabel.textColor = [NSColor systemRedColor];
        }
        return;
    }

    NSDictionary *payload = (NSDictionary *)parsed;
    NSString *errorText = jsonString(payload, @"error");
    if (errorText.length > 0) {
        if (gAnalyseStatusLabel != nil) {
            gAnalyseStatusLabel.hidden = NO;
            gAnalyseStatusLabel.stringValue = errorText;
            gAnalyseStatusLabel.textColor = [NSColor systemRedColor];
        }
        return;
    }

    NSString *verdict = jsonString(payload, @"verdict");
    NSString *summary = jsonString(payload, @"summary");
    NSString *explanation = jsonString(payload, @"explanation");
    NSString *provider = jsonString(payload, @"provider");

    if (gAnalyseStatusLabel != nil) {
        gAnalyseStatusLabel.hidden = YES;
        gAnalyseStatusLabel.stringValue = @"";
    }
    if (gVerdictBadge != nil) {
        gVerdictBadge.hidden = NO;
        gVerdictBadge.stringValue = verdictDisplayTitle(verdict);
        gVerdictBadge.textColor = verdictBadgeColor(verdict);
    }
    if (gAnalyseSummaryLabel != nil) {
        gAnalyseSummaryLabel.hidden = summary.length == 0;
        gAnalyseSummaryLabel.stringValue = summary;
    }
    if (gAnalyseExplanationView != nil && gAnalyseExplanationScroll != nil) {
        if (explanation.length > 0) {
            gAnalyseExplanationView.string = explanation;
            gAnalyseExplanationScroll.hidden = NO;
        } else {
            gAnalyseExplanationView.string = @"";
            gAnalyseExplanationScroll.hidden = YES;
        }
    }
    if (provider.length > 0 && gAnalyseStatusLabel != nil) {
        gAnalyseStatusLabel.hidden = NO;
        gAnalyseStatusLabel.stringValue = [NSString stringWithFormat:@"via %@", provider];
        gAnalyseStatusLabel.textColor = [NSColor tertiaryLabelColor];
    }
}

- (void)settingsMenuSelected:(id)sender {
    (void)sender;
    goOpenSettings();
}

- (void)settingsBackClicked:(id)sender {
    (void)sender;
    hideSettingsView();
}

- (void)settingsAddProvider:(id)sender {
    (void)sender;
    id target = [NSApp delegate];
    for (NSView *subview in [gSettingsRowsStack.arrangedSubviews copy]) {
        if (objc_getAssociatedObject(subview, "id_field") == nil) {
            [gSettingsRowsStack removeArrangedSubview:subview];
            [subview removeFromSuperview];
        }
    }
    NSDictionary *emptyRow = @{
        @"id": @"",
        @"driver": @"openai",
        @"model": @"",
        @"base_url": @"",
        @"api_key": @"",
        @"key_configured": @NO,
        @"is_default": @NO,
    };
    NSArray *drivers = objc_getAssociatedObject(gSettingsContainer, "drivers");
    if (![drivers isKindOfClass:[NSArray class]]) {
        drivers = @[];
    }
    NSUInteger rowIndex = gSettingsRowsStack.arrangedSubviews.count;
    NSView *rowView = makeSettingsProviderRow(emptyRow, drivers, target, rowIndex);
    [gSettingsRowsStack addArrangedSubview:rowView];
    [gSettingsRowsStack layoutSubtreeIfNeeded];
}

- (void)settingsSave:(id)sender {
    (void)sender;
    NSDictionary *payload = collectSettingsSavePayload();
    if (payload == nil) {
        return;
    }
    NSError *err = nil;
    NSData *data = [NSJSONSerialization dataWithJSONObject:payload options:0 error:&err];
    if (err != nil || data == nil) {
        return;
    }
    NSString *jsonStr = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    char *jsonCopy = copyCString(jsonStr);
    if (jsonCopy == NULL) {
        return;
    }
    goSaveSettings(jsonCopy);
    free(jsonCopy);
}

- (void)settingsRemoveRow:(NSButton *)sender {
    NSView *rowView = sender.superview;
    while (rowView != nil && rowView.superview != gSettingsRowsStack) {
        rowView = rowView.superview;
    }
    if (rowView == nil) {
        return;
    }
    [gSettingsRowsStack removeArrangedSubview:rowView];
    [rowView removeFromSuperview];
}

- (void)settingsDefaultChanged:(NSButton *)sender {
    if (sender.state != NSControlStateValueOn) {
        return;
    }
    for (NSView *rowView in gSettingsRowsStack.arrangedSubviews) {
        NSButton *radio = objc_getAssociatedObject(rowView, "default_radio");
        if (radio == nil || radio == sender) {
            continue;
        }
        radio.state = NSControlStateValueOff;
    }
}

- (void)showSettingsPanel:(NSString *)jsonStr {
    if (gSettingsContainer == nil || gBodyScroll == nil || jsonStr.length == 0) {
        return;
    }
    NSData *data = [jsonStr dataUsingEncoding:NSUTF8StringEncoding];
    NSError *err = nil;
    id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:&err];
    if (err != nil || ![parsed isKindOfClass:[NSDictionary class]]) {
        return;
    }
    NSDictionary *payload = (NSDictionary *)parsed;
    rebuildSettingsRowsFromJSON(payload, self);

    if (gPanelHeightConstraint != nil) {
        gPanelHeightConstraint.constant = kPanelHeightList;
    }
    gBodyContainer.hidden = NO;
    gBodyScroll.hidden = YES;
    gCommandReviewContainer.hidden = YES;
    gListHeaderStack.hidden = YES;
    gSettingsContainer.hidden = NO;
    gShowingCommandReview = NO;
    gShowingSettings = YES;
    gReviewRowID = nil;
    gReviewUseEventID = NO;
    clearAnalyseUI();
}

- (void)hideSettingsPanel {
    hideSettingsView();
}

- (void)settingsSaveError:(NSString *)message {
    NSAlert *alert = [[NSAlert alloc] init];
    alert.alertStyle = NSAlertStyleWarning;
    alert.messageText = @"Could not save settings";
    alert.informativeText = message ?: @"Unknown error";
    [alert addButtonWithTitle:@"OK"];
    [alert runModal];
}

- (void)popoverDidClose:(NSNotification *)notification {
    (void)notification;
    hideSettingsView();
}

- (void)rowDetailClicked:(NSClickGestureRecognizer *)sender {
    if (sender.state != NSGestureRecognizerStateEnded) {
        return;
    }
    NSView *view = sender.view;
    if (view == nil || gCachedTrayPayload == nil) {
        return;
    }
    NSNumber *indexNum = objc_getAssociatedObject(view, &kRowDetailIndexKey);
    if (indexNum == nil) {
        return;
    }
    enterCommandReviewHistory(gCachedTrayPayload, indexNum.integerValue);
}

- (void)setTrayIcon:(NSImage *)image {
    if (gStatusItem == nil || image == nil) {
        return;
    }
    image.size = NSMakeSize(18.0, 18.0);
    [image setTemplate:YES];
    gStatusItem.button.image = image;
}

- (void)setHeaderLogo:(NSImage *)image {
    if (gHeaderLogoView == nil || image == nil) {
        return;
    }
    image.size = NSMakeSize(24.0, 24.0);
    gHeaderLogoView.image = image;
}

- (void)setTrayTitle:(NSString *)title {
    if (gStatusItem == nil) {
        return;
    }
    gStatusItem.button.title = title ?: @"";
}

- (void)setTrayTooltip:(NSString *)tooltip {
    if (gStatusItem == nil) {
        return;
    }
    gStatusItem.button.toolTip = tooltip ?: @"";
}

- (void)showTrayPopover {
    if (gPopover == nil || gStatusItem == nil || gPopover.isShown) {
        return;
    }
    NSStatusBarButton *button = gStatusItem.button;
    [gPopover showRelativeToRect:button.bounds
                          ofView:button
                   preferredEdge:NSMinYEdge];
}

- (void)updateTrayPanel:(NSString *)jsonStr {
    if (gFooterDaemonLabel == nil || gBodyStack == nil || jsonStr.length == 0) {
        return;
    }

    NSData *data = [jsonStr dataUsingEncoding:NSUTF8StringEncoding];
    NSError *err = nil;
    id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:&err];
    if (err != nil || ![parsed isKindOfClass:[NSDictionary class]]) {
        return;
    }
    NSDictionary *payload = (NSDictionary *)parsed;

    gCachedTrayPayload = payload;

    gHasSavedListScroll = gPopover != nil && gPopover.isShown && !gShowingCommandReview && !gShowingSettings &&
                          pendingRowsFromPayload(payload).count == 0;
    if (gHasSavedListScroll && gBodyScroll != nil) {
        NSClipView *clip = gBodyScroll.contentView;
        if (clip != nil) {
            gSavedListScrollOrigin = clip.bounds.origin;
        }
    }

    gHistoryHasMore = jsonBool(payload, @"history_has_more");

    NSString *footerDaemon = jsonString(payload, @"footer_daemon");
    if (![footerDaemon isEqualToString:gFooterDaemonLabel.stringValue ?: @""]) {
        gFooterDaemonLabel.stringValue = footerDaemon;
    }
    NSString *footerPending = jsonString(payload, @"footer_pending");
    if (![footerPending isEqualToString:gFooterPendingLabel.stringValue ?: @""]) {
        gFooterPendingLabel.stringValue = footerPending;
    }

    NSInteger modeIndex = jsonInt(payload, @"mode_index");
    BOOL modeEnabled = jsonBool(payload, @"mode_enabled");
    if (gModeMenuButton != nil) {
        gModeMenuButton.enabled = modeEnabled;
    }
    if (gModeMenu != nil && modeIndex >= 0 && (NSUInteger)modeIndex < gModeMenu.itemArray.count) {
        [gModeMenu.itemArray[(NSUInteger)modeIndex] setState:NSControlStateValueOn];
        for (NSUInteger i = 0; i < gModeMenu.itemArray.count; i++) {
            if ((NSInteger)i != modeIndex) {
                [gModeMenu.itemArray[i] setState:NSControlStateValueOff];
            }
        }
    }

    BOOL updateVisible = jsonBool(payload, @"update_visible");
    if (gUpdateFooter != nil) {
        gUpdateFooter.hidden = !updateVisible;
    }
    if (updateVisible) {
        if (gUpdateLabel != nil) {
            gUpdateLabel.stringValue = jsonString(payload, @"update_label");
        }
        if (gInstallBtn != nil) {
            gInstallBtn.enabled = jsonBool(payload, @"update_enabled");
        }
    }

    rebuildBodyFromJSON(payload, self);

    if (gHasSavedListScroll && gBodyScroll != nil && !gBodyScroll.hidden) {
        NSClipView *clip = gBodyScroll.contentView;
        if (clip != nil) {
            restoreBodyScrollOrigin(clip, gSavedListScrollOrigin);
        }
    }
}

- (void)terminateTray {
    [NSApp terminate:nil];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    (void)sender;
    return NO;
}

@end

static void darwin_on_main(SEL selector, id object) {
    id target = [NSApp delegate];
    if (target == nil) {
        return;
    }
    [target performSelectorOnMainThread:selector
                             withObject:object
                          waitUntilDone:YES];
}

static BOOL isEffectiveAppearanceDark(NSView *view) {
    if (view == nil) {
        return NO;
    }
    NSAppearanceName match = [view.effectiveAppearance bestMatchFromAppearancesWithNames:@[
        NSAppearanceNameAqua,
        NSAppearanceNameDarkAqua,
    ]];
    return [match isEqualToString:NSAppearanceNameDarkAqua];
}

static void applySemanticTextColors(void) {
    if (gHeaderTitleLabel != nil) {
        gHeaderTitleLabel.textColor = [NSColor labelColor];
    }
    if (gReviewCommandView != nil) {
        gReviewCommandView.textColor = [NSColor labelColor];
    }
    if (gAnalyseExplanationView != nil) {
        gAnalyseExplanationView.textColor = [NSColor secondaryLabelColor];
    }
}

static void styleColoredActionButton(NSButton *btn, NSString *title, NSColor *accentColor) {
    if (btn == nil || title.length == 0 || accentColor == nil) {
        return;
    }
    btn.bezelStyle = NSBezelStyleRounded;
    btn.wantsLayer = YES;
    btn.layer.cornerRadius = 6.0;
    btn.layer.borderWidth = 1.0;
    btn.layer.borderColor = accentColor.CGColor;
    btn.layer.backgroundColor = [accentColor colorWithAlphaComponent:0.18].CGColor;
    NSMutableAttributedString *attr = [[NSMutableAttributedString alloc] initWithString:title];
    NSRange range = NSMakeRange(0, title.length);
    [attr addAttribute:NSForegroundColorAttributeName value:accentColor range:range];
    [attr addAttribute:NSFontAttributeName
                 value:[NSFont systemFontOfSize:[NSFont systemFontSize] weight:NSFontWeightSemibold]
                 range:range];
    btn.attributedTitle = attr;
}

static NSArray *historyRowsFromPayload(NSDictionary *payload) {
    if (payload == nil) {
        return @[];
    }
    NSArray *rows = payload[@"history_rows"];
    if (![rows isKindOfClass:[NSArray class]]) {
        return @[];
    }
    return rows;
}

static void clearAnalyseUI(void) {
    gAnalyseInFlight = NO;
    if (gAnalyseBtn != nil) {
        gAnalyseBtn.enabled = YES;
    }
    if (gAnalyseStatusLabel != nil) {
        gAnalyseStatusLabel.hidden = YES;
        gAnalyseStatusLabel.stringValue = @"";
        gAnalyseStatusLabel.textColor = [NSColor secondaryLabelColor];
    }
    if (gVerdictBadge != nil) {
        gVerdictBadge.hidden = YES;
        gVerdictBadge.stringValue = @"";
    }
    if (gAnalyseSummaryLabel != nil) {
        gAnalyseSummaryLabel.hidden = YES;
        gAnalyseSummaryLabel.stringValue = @"";
    }
    if (gAnalyseExplanationScroll != nil) {
        gAnalyseExplanationScroll.hidden = YES;
    }
    if (gAnalyseExplanationView != nil) {
        gAnalyseExplanationView.string = @"";
    }
}

static NSString *verdictDisplayTitle(NSString *verdict) {
    if ([verdict isEqualToString:@"safe"]) {
        return @"Safe";
    }
    if ([verdict isEqualToString:@"caution"]) {
        return @"Caution";
    }
    if ([verdict isEqualToString:@"dangerous"]) {
        return @"Dangerous";
    }
    return @"Unknown";
}

static NSColor *verdictBadgeColor(NSString *verdict) {
    if ([verdict isEqualToString:@"safe"]) {
        return [NSColor systemGreenColor];
    }
    if ([verdict isEqualToString:@"caution"]) {
        return [NSColor systemOrangeColor];
    }
    if ([verdict isEqualToString:@"dangerous"]) {
        return [NSColor systemRedColor];
    }
    return [NSColor secondaryLabelColor];
}

static void notifyAppearanceChanged(void) {
    applySemanticTextColors();
    goAppearanceChanged(isEffectiveAppearanceDark(gPanelRoot) ? 1 : 0);
}

static NSString *jsonString(id dict, NSString *key) {
    id value = dict[key];
    if (value == nil || value == [NSNull null]) {
        return @"";
    }
    if ([value isKindOfClass:[NSString class]]) {
        return value;
    }
    return [value description];
}

static BOOL jsonBool(id dict, NSString *key) {
    id value = dict[key];
    if ([value isKindOfClass:[NSNumber class]]) {
        return [value boolValue];
    }
    return NO;
}

static NSInteger jsonInt(id dict, NSString *key) {
    id value = dict[key];
    if ([value isKindOfClass:[NSNumber class]]) {
        return [value integerValue];
    }
    return 0;
}

static void attachRowDetailTap(NSView *view, NSInteger rowIndex, id target) {
    if (view == nil) {
        return;
    }
    objc_setAssociatedObject(view, &kRowDetailIndexKey, @(rowIndex), OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    for (NSGestureRecognizer *existing in [view.gestureRecognizers copy]) {
        if ([existing isKindOfClass:[NSClickGestureRecognizer class]]) {
            [view removeGestureRecognizer:existing];
        }
    }
    NSClickGestureRecognizer *tap = [[NSClickGestureRecognizer alloc] initWithTarget:target
                                                                              action:@selector(rowDetailClicked:)];
    tap.numberOfClicksRequired = 1;
    [view addGestureRecognizer:tap];
}

static NSView *makeLoadMoreRow(id target) {
    CGFloat rowWidth = kPanelWidth - (kPadding * 2);
    NSView *row = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, rowWidth, 28.0)];
    row.translatesAutoresizingMaskIntoConstraints = NO;

    NSTextField *label = [NSTextField labelWithString:@"Load more…"];
    label.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    label.textColor = [NSColor linkColor];
    label.translatesAutoresizingMaskIntoConstraints = NO;
    [row addSubview:label];

    NSClickGestureRecognizer *tap = [[NSClickGestureRecognizer alloc] initWithTarget:target
                                                                              action:@selector(loadMoreClicked:)];
    tap.numberOfClicksRequired = 1;
    [row addGestureRecognizer:tap];

    [NSLayoutConstraint activateConstraints:@[
        [label.centerXAnchor constraintEqualToAnchor:row.centerXAnchor],
        [label.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [row.heightAnchor constraintEqualToConstant:28.0],
        [row.widthAnchor constraintEqualToConstant:rowWidth],
    ]];
    return row;
}

static NSView *makeHistoryRow(NSString *label, NSString *detail, NSString *rowID, NSInteger rowIndex, id target) {
    (void)detail;
    (void)rowID;
    CGFloat rowWidth = kPanelWidth - (kPadding * 2);

    NSView *row = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, rowWidth, kRowHeight)];
    row.translatesAutoresizingMaskIntoConstraints = NO;

    NSTextField *labelField = [NSTextField labelWithString:label];
    labelField.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    labelField.textColor = [NSColor labelColor];
    labelField.lineBreakMode = NSLineBreakByTruncatingTail;
    labelField.translatesAutoresizingMaskIntoConstraints = NO;
    [row addSubview:labelField];

    [NSLayoutConstraint activateConstraints:@[
        [row.widthAnchor constraintEqualToConstant:rowWidth],
        [labelField.leadingAnchor constraintEqualToAnchor:row.leadingAnchor],
        [labelField.trailingAnchor constraintEqualToAnchor:row.trailingAnchor],
        [labelField.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [row.heightAnchor constraintEqualToConstant:kRowHeight],
    ]];

    attachRowDetailTap(row, rowIndex, target);
    return row;
}

static void addRowsFromJSONArray(NSArray *rows, id target, NSMutableArray *outViews) {
    if (![rows isKindOfClass:[NSArray class]]) {
        return;
    }
    for (NSUInteger i = 0; i < rows.count; i++) {
        id rowObj = rows[i];
        if (![rowObj isKindOfClass:[NSDictionary class]]) {
            continue;
        }
        NSDictionary *row = (NSDictionary *)rowObj;
        NSString *rowID = jsonString(row, @"id");
        NSString *rowLabel = jsonString(row, @"label");
        NSString *rowDetail = jsonString(row, @"detail");
        if (rowID.length == 0) {
            continue;
        }
        [outViews addObject:makeHistoryRow(rowLabel, rowDetail, rowID, (NSInteger)i, target)];
    }
}

static NSString *detailTextForRowID(NSDictionary *payload, NSString *rowID) {
    if (rowID.length == 0) {
        return nil;
    }
    NSArray *keys = @[@"pending_rows", @"history_rows"];
    for (NSString *key in keys) {
        NSArray *rows = payload[key];
        if (![rows isKindOfClass:[NSArray class]]) {
            continue;
        }
        for (id rowObj in rows) {
            if (![rowObj isKindOfClass:[NSDictionary class]]) {
                continue;
            }
            NSDictionary *row = (NSDictionary *)rowObj;
            if (![rowID isEqualToString:jsonString(row, @"id")]) {
                continue;
            }
            NSString *detail = jsonString(row, @"detail");
            NSString *label = jsonString(row, @"label");
            return detail.length > 0 ? detail : label;
        }
    }
    return nil;
}

static void hideSettingsView(void) {
    if (gSettingsContainer == nil) {
        return;
    }
    gSettingsContainer.hidden = YES;
    gShowingSettings = NO;
    if (gShowingCommandReview) {
        applyPanelLayoutForReviewMode(YES);
    } else {
        applyPanelLayoutForReviewMode(NO);
        if (gListHeaderStack != nil) {
            gListHeaderStack.hidden = NO;
        }
        if (gBodyScroll != nil) {
            gBodyScroll.hidden = NO;
        }
    }
}

static void populateDriverPopup(NSPopUpButton *popup, NSArray *drivers, NSString *selected) {
    [popup removeAllItems];
    NSString *selectedName = selected.length > 0 ? selected : @"openai";
    NSInteger selectIndex = 0;
    for (NSUInteger i = 0; i < drivers.count; i++) {
        id driverObj = drivers[i];
        if (![driverObj isKindOfClass:[NSDictionary class]]) {
            continue;
        }
        NSDictionary *driver = (NSDictionary *)driverObj;
        NSString *name = jsonString(driver, @"name");
        NSString *label = jsonString(driver, @"label");
        if (name.length == 0) {
            continue;
        }
        if (label.length == 0) {
            label = name;
        }
        [popup addItemWithTitle:label];
        [[popup lastItem] setRepresentedObject:name];
        if ([name isEqualToString:selectedName]) {
            selectIndex = (NSInteger)[popup numberOfItems] - 1;
        }
    }
    if ([popup numberOfItems] > 0) {
        [popup selectItemAtIndex:selectIndex];
    }
}

static NSTextField *makeSettingsLabel(NSString *text) {
    NSTextField *field = [NSTextField labelWithString:text];
    field.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    field.textColor = [NSColor secondaryLabelColor];
    return field;
}

static NSTextField *makeSettingsInput(NSString *placeholder) {
    NSTextField *field = [[NSTextField alloc] init];
    field.translatesAutoresizingMaskIntoConstraints = NO;
    field.placeholderString = placeholder ?: @"";
    field.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    return field;
}

static NSView *makeSettingsProviderRow(NSDictionary *row, NSArray *drivers, id target, NSUInteger rowIndex) {
    CGFloat rowWidth = kPanelWidth - (kPadding * 2) - 8.0;

    NSView *card = [[NSView alloc] init];
    card.translatesAutoresizingMaskIntoConstraints = NO;
    card.wantsLayer = YES;
    card.layer.backgroundColor = [[NSColor separatorColor] colorWithAlphaComponent:0.25].CGColor;
    card.layer.cornerRadius = 6.0;

    NSStackView *cardStack = [[NSStackView alloc] init];
    cardStack.translatesAutoresizingMaskIntoConstraints = NO;
    cardStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    cardStack.alignment = NSLayoutAttributeLeading;
    cardStack.spacing = 4.0;
    [card addSubview:cardStack];
    [NSLayoutConstraint activateConstraints:@[
        [cardStack.topAnchor constraintEqualToAnchor:card.topAnchor constant:8.0],
        [cardStack.bottomAnchor constraintEqualToAnchor:card.bottomAnchor constant:-8.0],
        [cardStack.leadingAnchor constraintEqualToAnchor:card.leadingAnchor constant:8.0],
        [cardStack.trailingAnchor constraintEqualToAnchor:card.trailingAnchor constant:-8.0],
        [card.widthAnchor constraintEqualToConstant:rowWidth],
    ]];

    NSStackView *headerRow = [[NSStackView alloc] init];
    headerRow.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    headerRow.alignment = NSLayoutAttributeCenterY;
    headerRow.spacing = 8.0;
    [cardStack addArrangedSubview:headerRow];
    [headerRow.widthAnchor constraintEqualToConstant:rowWidth - 16.0].active = YES;

    NSTextField *idLabel = makeSettingsLabel(@"ID");
    [cardStack addArrangedSubview:idLabel];
    NSTextField *idField = makeSettingsInput(@"provider-id");
    idField.stringValue = jsonString(row, @"id");
    [cardStack addArrangedSubview:idField];
    [idField.widthAnchor constraintEqualToConstant:rowWidth - 16.0].active = YES;
    objc_setAssociatedObject(card, "id_field", idField, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSTextField *driverLabel = makeSettingsLabel(@"Driver");
    [cardStack addArrangedSubview:driverLabel];
    NSPopUpButton *driverPopup = [[NSPopUpButton alloc] init];
    driverPopup.translatesAutoresizingMaskIntoConstraints = NO;
    populateDriverPopup(driverPopup, drivers, jsonString(row, @"driver"));
    [cardStack addArrangedSubview:driverPopup];
    objc_setAssociatedObject(card, "driver_popup", driverPopup, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSTextField *modelLabel = makeSettingsLabel(@"Model");
    [cardStack addArrangedSubview:modelLabel];
    NSTextField *modelField = makeSettingsInput(@"gpt-4o-mini");
    modelField.stringValue = jsonString(row, @"model");
    [cardStack addArrangedSubview:modelField];
    [modelField.widthAnchor constraintEqualToConstant:rowWidth - 16.0].active = YES;
    objc_setAssociatedObject(card, "model_field", modelField, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSTextField *baseLabel = makeSettingsLabel(@"Base URL");
    [cardStack addArrangedSubview:baseLabel];
    NSTextField *baseField = makeSettingsInput(@"https://api.openai.com/v1");
    baseField.stringValue = jsonString(row, @"base_url");
    [cardStack addArrangedSubview:baseField];
    [baseField.widthAnchor constraintEqualToConstant:rowWidth - 16.0].active = YES;
    objc_setAssociatedObject(card, "base_field", baseField, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSTextField *keyLabel = makeSettingsLabel(@"API Key");
    [cardStack addArrangedSubview:keyLabel];
    NSSecureTextField *keyField = [[NSSecureTextField alloc] init];
    keyField.translatesAutoresizingMaskIntoConstraints = NO;
    keyField.placeholderString = jsonBool(row, @"key_configured") ? @"Leave blank to keep" : @"sk-…";
    keyField.stringValue = jsonString(row, @"api_key");
    keyField.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    [cardStack addArrangedSubview:keyField];
    [keyField.widthAnchor constraintEqualToConstant:rowWidth - 16.0].active = YES;
    objc_setAssociatedObject(card, "key_field", keyField, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSButton *defaultRadio = [[NSButton alloc] init];
    [defaultRadio setButtonType:NSButtonTypeRadio];
    defaultRadio.title = @"Default provider";
    defaultRadio.target = target;
    defaultRadio.action = @selector(settingsDefaultChanged:);
    defaultRadio.state = jsonBool(row, @"is_default") ? NSControlStateValueOn : NSControlStateValueOff;
    [cardStack addArrangedSubview:defaultRadio];
    objc_setAssociatedObject(card, "default_radio", defaultRadio, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    NSView *headerSpacer = [[NSView alloc] init];
    [headerSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationHorizontal];
    [headerRow addArrangedSubview:headerSpacer];

    NSButton *removeBtn = [NSButton buttonWithTitle:@"Remove" target:target action:@selector(settingsRemoveRow:)];
    removeBtn.bezelStyle = NSBezelStyleRounded;
    removeBtn.tag = (NSInteger)rowIndex;
    [headerRow addArrangedSubview:removeBtn];

    (void)rowIndex;
    return card;
}

static void rebuildSettingsRowsFromJSON(NSDictionary *payload, id target) {
    NSArray *drivers = payload[@"drivers"];
    if (![drivers isKindOfClass:[NSArray class]]) {
        drivers = @[];
    }
    objc_setAssociatedObject(gSettingsContainer, "drivers", drivers, OBJC_ASSOCIATION_RETAIN_NONATOMIC);

    for (NSView *subview in [gSettingsRowsStack.arrangedSubviews copy]) {
        [gSettingsRowsStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }

    NSArray *providers = payload[@"providers"];
    if ([providers isKindOfClass:[NSArray class]] && providers.count > 0) {
        for (NSUInteger i = 0; i < providers.count; i++) {
            id rowObj = providers[i];
            if (![rowObj isKindOfClass:[NSDictionary class]]) {
                continue;
            }
            NSView *rowView = makeSettingsProviderRow((NSDictionary *)rowObj, drivers, target, i);
            [gSettingsRowsStack addArrangedSubview:rowView];
        }
    } else {
        NSTextField *empty = [NSTextField labelWithString:@"No providers configured. Add one below."];
        empty.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
        empty.textColor = [NSColor secondaryLabelColor];
        empty.alignment = NSTextAlignmentCenter;
        [gSettingsRowsStack addArrangedSubview:empty];
    }

    [gSettingsRowsStack layoutSubtreeIfNeeded];
}

static NSString *selectedDriverName(NSPopUpButton *popup) {
    if (popup == nil || [popup numberOfItems] == 0) {
        return @"openai";
    }
    NSString *name = [[popup selectedItem] representedObject];
    if ([name isKindOfClass:[NSString class]] && [(NSString *)name length] > 0) {
        return name;
    }
    return @"openai";
}

static NSDictionary *collectSettingsSavePayload(void) {
    NSMutableArray *providers = [NSMutableArray array];
    NSString *defaultProvider = @"";

    for (NSView *rowView in gSettingsRowsStack.arrangedSubviews) {
        NSTextField *idField = objc_getAssociatedObject(rowView, "id_field");
        if (idField == nil) {
            continue;
        }
        NSPopUpButton *driverPopup = objc_getAssociatedObject(rowView, "driver_popup");
        NSTextField *modelField = objc_getAssociatedObject(rowView, "model_field");
        NSTextField *baseField = objc_getAssociatedObject(rowView, "base_field");
        NSSecureTextField *keyField = objc_getAssociatedObject(rowView, "key_field");
        NSButton *defaultRadio = objc_getAssociatedObject(rowView, "default_radio");

        BOOL isDefault = defaultRadio != nil && defaultRadio.state == NSControlStateValueOn;
        NSString *rowID = idField.stringValue ?: @"";
        if (isDefault) {
            defaultProvider = rowID;
        }

        [providers addObject:@{
            @"id": rowID,
            @"driver": selectedDriverName(driverPopup),
            @"model": modelField.stringValue ?: @"",
            @"base_url": baseField.stringValue ?: @"",
            @"api_key": keyField.stringValue ?: @"",
            @"is_default": @(isDefault),
        }];
    }

    return @{
        @"providers": providers,
        @"default_provider": defaultProvider,
    };
}

static NSArray *pendingRowsFromPayload(NSDictionary *payload) {
    if (payload == nil) {
        return @[];
    }
    NSArray *rows = payload[@"pending_rows"];
    if (![rows isKindOfClass:[NSArray class]]) {
        return @[];
    }
    return rows;
}

static void clearListBodyStack(void) {
    if (gBodyStack == nil) {
        return;
    }
    for (NSView *subview in [gBodyStack.arrangedSubviews copy]) {
        [gBodyStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }
}

static void applyPanelLayoutForReviewMode(BOOL reviewMode) {
    if (gPanelHeightConstraint != nil) {
        gPanelHeightConstraint.constant = reviewMode ? kPanelHeightReview : kPanelHeightList;
    }
    if (gBodyScroll != nil) {
        gBodyScroll.hidden = reviewMode;
    }
    if (gCommandReviewContainer != nil) {
        gCommandReviewContainer.hidden = !reviewMode;
    }
    if (gListHeaderStack != nil) {
        gListHeaderStack.hidden = reviewMode;
    }
    if (gUpdateFooter != nil && reviewMode) {
        gUpdateFooter.hidden = YES;
    }
    if (gReviewBackBtn != nil) {
        gReviewBackBtn.hidden = !reviewMode || gReviewMode != CommandReviewModeHistory;
    }
    if (gReviewCenterActions != nil) {
        gReviewCenterActions.hidden = !reviewMode || gReviewMode != CommandReviewModePending;
    }
    if (reviewMode && gReviewMode == CommandReviewModePending && gHeaderTitleLabel != nil) {
        gHeaderTitleLabel.stringValue = @"Pending approval";
    } else if (gHeaderTitleLabel != nil) {
        gHeaderTitleLabel.stringValue = @"SideGuard";
    }
    if (reviewMode) {
        [gPanelRoot layoutSubtreeIfNeeded];
    }
}

static void exitCommandReview(void) {
    BOOL leavingPending = gReviewMode == CommandReviewModePending;
    clearAnalyseUI();
    gShowingCommandReview = NO;
    gReviewRows = nil;
    gReviewIndex = 0;
    gReviewSkipMode = NO;
    gReviewDecided = NO;
    gReviewRowID = nil;
    gReviewUseEventID = NO;
    applyPanelLayoutForReviewMode(NO);
    if (leavingPending) {
        gHasSavedListScroll = NO;
    }
    if (gBodyScroll != nil) {
        gBodyScroll.hidden = NO;
    }
    if (leavingPending) {
        resetBodyScrollToTop();
    }
}

static NSDictionary *reviewRowAtIndex(NSArray *rows, NSInteger index) {
    if (rows == nil || index < 0 || (NSUInteger)index >= rows.count) {
        return nil;
    }
    id rowObj = rows[(NSUInteger)index];
    if (![rowObj isKindOfClass:[NSDictionary class]]) {
        return nil;
    }
    return (NSDictionary *)rowObj;
}

static void updateCommandReviewDots(void) {
    if (gReviewDotsStack == nil) {
        return;
    }
    for (NSView *subview in [gReviewDotsStack.arrangedSubviews copy]) {
        [gReviewDotsStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }
    NSUInteger count = gReviewRows.count;
    if (count <= 1) {
        return;
    }
    for (NSUInteger i = 0; i < count; i++) {
        NSView *dot = [[NSView alloc] init];
        dot.translatesAutoresizingMaskIntoConstraints = NO;
        dot.wantsLayer = YES;
        CGFloat size = (NSInteger)i == gReviewIndex ? 8.0 : 6.0;
        dot.layer.cornerRadius = size / 2.0;
        BOOL active = (NSInteger)i == gReviewIndex;
        dot.layer.backgroundColor = active
            ? (gReviewMode == CommandReviewModePending
                   ? [NSColor systemOrangeColor].CGColor
                   : [NSColor controlAccentColor].CGColor)
            : [[NSColor secondaryLabelColor] colorWithAlphaComponent:0.45].CGColor;
        [gReviewDotsStack addArrangedSubview:dot];
        [NSLayoutConstraint activateConstraints:@[
            [dot.widthAnchor constraintEqualToConstant:size],
            [dot.heightAnchor constraintEqualToConstant:size],
        ]];
    }
}

static void syncCommandReviewNavButtons(void) {
    NSUInteger count = gReviewRows.count;
    BOOL hasPrev = NO;
    BOOL hasNext = NO;

    if (gReviewMode == CommandReviewModePending) {
        hasPrev = gReviewIndex > 0 && !gReviewDecided;
        hasNext = count > 0 && (NSUInteger)gReviewIndex < count - 1 && !gReviewSkipMode && !gReviewDecided;
    } else {
        hasPrev = gReviewIndex > 0;
        hasNext = count > 0 && (NSUInteger)gReviewIndex < count - 1;
    }

    if (gReviewPrevBtn != nil) {
        gReviewPrevBtn.enabled = hasPrev;
    }
    if (gReviewNextBtn != nil) {
        gReviewNextBtn.enabled = hasNext;
    }
    if (gReviewRunBtn != nil) {
        gReviewRunBtn.enabled = gReviewMode == CommandReviewModePending && !gReviewDecided && count > 0;
    }
    if (gReviewDeclineBtn != nil) {
        gReviewDeclineBtn.enabled = gReviewMode == CommandReviewModePending && !gReviewDecided && count > 0;
    }
}

static void refreshCommandReviewSlide(NSDictionary *payload) {
    if (gReviewRows.count == 0) {
        return;
    }
    if (gReviewIndex < 0) {
        gReviewIndex = 0;
    }
    if ((NSUInteger)gReviewIndex >= gReviewRows.count) {
        gReviewIndex = (NSInteger)gReviewRows.count - 1;
    }

    NSDictionary *row = reviewRowAtIndex(gReviewRows, gReviewIndex);
    if (row == nil) {
        return;
    }

    NSString *rowID = jsonString(row, @"id");
    NSString *detail = jsonString(row, @"detail");
    NSString *label = jsonString(row, @"label");
    NSString *commandText = detail.length > 0 ? detail : label;
    if (payload != nil && rowID.length > 0) {
        NSString *updated = detailTextForRowID(payload, rowID);
        if (updated.length > 0) {
            commandText = updated;
        }
    }

    BOOL slideChanged = gReviewRowID == nil || ![rowID isEqualToString:gReviewRowID];
    if (slideChanged) {
        clearAnalyseUI();
    }

    gReviewRowID = rowID.length > 0 ? [rowID copy] : nil;
    gReviewUseEventID = gReviewMode == CommandReviewModeHistory;

    if (gReviewCommandView != nil) {
        gReviewCommandView.string = commandText ?: @"";
        [gReviewCommandView scrollPoint:NSMakePoint(0, 0)];
    }
    if (gReviewRunBtn != nil) {
        gReviewRunBtn.identifier = rowID;
    }
    if (gReviewDeclineBtn != nil) {
        gReviewDeclineBtn.identifier = rowID;
    }
    if (gReviewCounter != nil) {
        gReviewCounter.stringValue =
            [NSString stringWithFormat:@"%ld of %lu",
             (long)gReviewIndex + 1,
             (unsigned long)gReviewRows.count];
    }

    updateCommandReviewDots();
    syncCommandReviewNavButtons();
}

static void enterCommandReviewHistory(NSDictionary *payload, NSInteger index) {
    NSArray *rows = historyRowsFromPayload(payload);
    if (rows.count == 0) {
        return;
    }
    gReviewMode = CommandReviewModeHistory;
    gReviewRows = rows;
    gReviewIndex = MAX(0, MIN(index, (NSInteger)rows.count - 1));
    gReviewSkipMode = NO;
    gReviewDecided = NO;
    gShowingCommandReview = YES;
    applyPanelLayoutForReviewMode(YES);
    refreshCommandReviewSlide(payload);
}

static void enterCommandReviewPending(NSDictionary *payload, id target) {
    (void)target;
    NSArray *rows = pendingRowsFromPayload(payload);
    if (rows.count == 0) {
        if (gShowingCommandReview && gReviewMode == CommandReviewModePending) {
            exitCommandReview();
        }
        return;
    }

    NSString *previousID = nil;
    if (gReviewMode == CommandReviewModePending && gShowingCommandReview) {
        NSDictionary *previousRow = reviewRowAtIndex(gReviewRows, gReviewIndex);
        if (previousRow != nil) {
            previousID = jsonString(previousRow, @"id");
        }
    } else {
        clearListBodyStack();
    }

    gReviewMode = CommandReviewModePending;
    gReviewRows = rows;
    gShowingCommandReview = YES;
    gReviewUseEventID = NO;
    gHasSavedListScroll = NO;

    NSInteger newIndex = 0;
    BOOL foundPrevious = NO;
    if (previousID.length > 0) {
        for (NSUInteger i = 0; i < rows.count; i++) {
            id rowObj = rows[i];
            if (![rowObj isKindOfClass:[NSDictionary class]]) {
                continue;
            }
            if ([previousID isEqualToString:jsonString((NSDictionary *)rowObj, @"id")]) {
                newIndex = (NSInteger)i;
                foundPrevious = YES;
                break;
            }
        }
    } else if (gReviewIndex >= 0 && (NSUInteger)gReviewIndex < rows.count) {
        newIndex = gReviewIndex;
        foundPrevious = YES;
    }

    if (!foundPrevious) {
        if (gReviewDecided && gReviewIndex >= 0 && (NSUInteger)gReviewIndex < rows.count) {
            newIndex = gReviewIndex;
        } else if ((NSUInteger)gReviewIndex >= rows.count) {
            newIndex = (NSInteger)rows.count - 1;
        }
        gReviewSkipMode = NO;
        gReviewDecided = NO;
    }

    gReviewIndex = newIndex;
    applyPanelLayoutForReviewMode(YES);
    refreshCommandReviewSlide(payload);
}

static NSString *bodyPayloadFingerprint(NSDictionary *payload) {
    if (payload == nil) {
        return @"";
    }
    NSArray *pendingRows = pendingRowsFromPayload(payload);
    if (pendingRows.count > 0) {
        NSDictionary *body = @{
            @"pending_rows": pendingRows,
            @"pending_overflow": payload[@"pending_overflow"] ?: @"",
        };
        NSData *data = [NSJSONSerialization dataWithJSONObject:body options:NSJSONWritingSortedKeys error:nil];
        if (data == nil) {
            return @"";
        }
        return [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding] ?: @"";
    }
    NSDictionary *body = @{
        @"pending_rows": @[],
        @"history_rows": payload[@"history_rows"] ?: @[],
        @"history_has_more": payload[@"history_has_more"] ?: @NO,
        @"pending_overflow": payload[@"pending_overflow"] ?: @"",
        @"empty_message": payload[@"empty_message"] ?: @"",
    };
    NSData *data = [NSJSONSerialization dataWithJSONObject:body options:NSJSONWritingSortedKeys error:nil];
    if (data == nil) {
        return @"";
    }
    return [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding] ?: @"";
}

static void restoreBodyScrollOrigin(NSClipView *clipView, NSPoint origin) {
    if (clipView == nil || gBodyScroll == nil) {
        return;
    }
    [clipView scrollToPoint:origin];
    [gBodyScroll reflectScrolledClipView:clipView];
    dispatch_async(dispatch_get_main_queue(), ^{
        [clipView scrollToPoint:origin];
        [gBodyScroll reflectScrolledClipView:clipView];
    });
}

static void resetBodyScrollToTop(void) {
    if (gBodyScroll == nil) {
        return;
    }
    NSClipView *clipView = gBodyScroll.contentView;
    if (clipView == nil) {
        return;
    }
    restoreBodyScrollOrigin(clipView, NSMakePoint(0, 0));
    gHasSavedListScroll = NO;
    gSavedListScrollOrigin = NSZeroPoint;
}

static void rebuildBodyFromJSON(NSDictionary *payload, id target) {
    BOOL wasShowingReview = gShowingCommandReview;
    CommandReviewMode previousReviewMode = gReviewMode;
    NSString *savedReviewRowID = wasShowingReview ? [gReviewRowID copy] : nil;
    NSInteger savedReviewIndex = gReviewIndex;
    BOOL wasShowingSettings = gShowingSettings;

    NSArray *pendingRows = pendingRowsFromPayload(payload);
    BOOL hasPending = pendingRows.count > 0;

    NSString *fingerprint = bodyPayloadFingerprint(payload);
    BOOL fingerprintMatch = gLastBodyFingerprint != nil && [fingerprint isEqualToString:gLastBodyFingerprint];
    NSClipView *clipViewBefore = gBodyScroll ? gBodyScroll.contentView : nil;

    if (fingerprintMatch) {
        if (gShowingCommandReview && !wasShowingSettings) {
            if (gReviewMode == CommandReviewModePending && hasPending) {
                gReviewRows = pendingRows;
            } else if (gReviewMode == CommandReviewModeHistory) {
                gReviewRows = historyRowsFromPayload(payload);
            }
            refreshCommandReviewSlide(payload);
        } else if (hasPending && !wasShowingSettings) {
            enterCommandReviewPending(payload, target);
        } else if (gHasSavedListScroll && clipViewBefore != nil) {
            restoreBodyScrollOrigin(clipViewBefore, gSavedListScrollOrigin);
        }
        return;
    }
    gLastBodyFingerprint = [fingerprint copy];

    if (hasPending && !wasShowingSettings) {
        enterCommandReviewPending(payload, target);
        gLastPendingRowCount = pendingRows.count;
        return;
    }

    BOOL leavingPendingReview = wasShowingReview && previousReviewMode == CommandReviewModePending;
    if (leavingPendingReview) {
        exitCommandReview();
        wasShowingReview = NO;
    }

    NSClipView *clipView = gBodyScroll ? gBodyScroll.contentView : nil;
    NSPoint savedOrigin = NSZeroPoint;
    CGFloat docHeightBefore = 0;
    NSUInteger pendingCountBefore = gLastPendingRowCount;
    BOOL preserveScroll = clipView != nil && !wasShowingReview && !wasShowingSettings && !leavingPendingReview &&
                          gHasSavedListScroll;
    if (preserveScroll) {
        savedOrigin = gSavedListScrollOrigin;
        if (clipView.documentView != nil) {
            docHeightBefore = NSHeight(clipView.documentView.bounds);
        }
    }

    for (NSView *subview in [gBodyStack.arrangedSubviews copy]) {
        [gBodyStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }

    NSString *overflow = jsonString(payload, @"pending_overflow");
    if (overflow.length > 0) {
        NSTextField *hint = [NSTextField labelWithString:overflow];
        hint.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
        hint.textColor = [NSColor secondaryLabelColor];
        [gBodyStack addArrangedSubview:hint];
    }

    NSArray *historyRows = historyRowsFromPayload(payload);
    if (historyRows.count > 0) {
        NSMutableArray *historyViews = [NSMutableArray array];
        addRowsFromJSONArray(historyRows, target, historyViews);
        for (NSView *view in historyViews) {
            [gBodyStack addArrangedSubview:view];
        }
    }

    if (jsonBool(payload, @"history_has_more")) {
        [gBodyStack addArrangedSubview:makeLoadMoreRow(target)];
    }

    NSString *emptyMsg = jsonString(payload, @"empty_message");
    if (emptyMsg.length > 0 && gBodyStack.arrangedSubviews.count == 0) {
        NSTextField *empty = [NSTextField labelWithString:emptyMsg];
        empty.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
        empty.textColor = [NSColor secondaryLabelColor];
        empty.alignment = NSTextAlignmentCenter;
        [gBodyStack addArrangedSubview:empty];
    }

    gLastPendingRowCount = 0;

    [gBodyStack layoutSubtreeIfNeeded];

    if (preserveScroll && clipView.documentView != nil) {
        CGFloat docHeightAfter = NSHeight(clipView.documentView.bounds);
        CGFloat heightDelta = docHeightAfter - docHeightBefore;
        if (pendingCountBefore > 0 && pendingRows.count > pendingCountBefore && savedOrigin.y > 0) {
            CGFloat pendingRowDelta = (pendingRows.count - pendingCountBefore) * (kRowHeight + 6.0);
            savedOrigin.y += pendingRowDelta;
        } else if (pendingCountBefore == 0 && pendingRows.count == 0 && heightDelta > 0) {
            // History-only list grew at the bottom — keep the same scroll origin.
        } else if (heightDelta > 0 && savedOrigin.y > 0 && pendingRows.count > pendingCountBefore) {
            savedOrigin.y += heightDelta;
        }
        NSRect docBounds = clipView.documentView.bounds;
        CGFloat maxY = MAX(0, NSMaxY(docBounds) - NSHeight(clipView.bounds));
        savedOrigin.y = MIN(MAX(0, savedOrigin.y), maxY);
        restoreBodyScrollOrigin(clipView, savedOrigin);
    } else if (leavingPendingReview) {
        resetBodyScrollToTop();
    }

    if (wasShowingReview && previousReviewMode == CommandReviewModeHistory && !wasShowingSettings) {
        NSInteger restoreIndex = savedReviewIndex;
        if (savedReviewRowID.length > 0) {
            NSArray *rows = historyRowsFromPayload(payload);
            for (NSUInteger i = 0; i < rows.count; i++) {
                NSDictionary *row = reviewRowAtIndex(rows, (NSInteger)i);
                if (row != nil && [savedReviewRowID isEqualToString:jsonString(row, @"id")]) {
                    restoreIndex = (NSInteger)i;
                    break;
                }
            }
        }
        enterCommandReviewHistory(payload, restoreIndex);
    } else if (wasShowingSettings) {
        gBodyScroll.hidden = YES;
        gCommandReviewContainer.hidden = YES;
        gSettingsContainer.hidden = NO;
        gShowingSettings = YES;
    }
}

void darwin_tray_prepare(darwin_ready_fn on_ready) {
    @autoreleasepool {
        gOnReady = on_ready;

        [NSApplication sharedApplication];
        gTrayDelegate = [[SideGuardTrayDelegate alloc] init];
        [NSApp setDelegate:gTrayDelegate];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    }
}

void darwin_tray_run_loop(void) {
    @autoreleasepool {
        // Workaround (mirrors getlantern/systray): [NSApp run] must be invoked from a
        // different translation unit than NSApplication setup to avoid AppKit crashes.
        [NSApp run];
    }
}

void darwin_set_icon(const unsigned char *data, size_t len) {
    if (data == NULL || len == 0) {
        return;
    }
    NSData *pngData = [NSData dataWithBytes:data length:len];
    NSImage *image = [[NSImage alloc] initWithData:pngData];
    if (image == nil) {
        return;
    }
    darwin_on_main(@selector(setTrayIcon:), image);
}

void darwin_set_header_logo(const unsigned char *data, size_t len) {
    if (data == NULL || len == 0) {
        return;
    }
    NSData *pngData = [NSData dataWithBytes:data length:len];
    NSImage *image = [[NSImage alloc] initWithData:pngData];
    if (image == nil) {
        return;
    }
    darwin_on_main(@selector(setHeaderLogo:), image);
}

void darwin_set_title(const char *title) {
    NSString *nsTitle = (title != NULL) ? [NSString stringWithUTF8String:title] : @"";
    darwin_on_main(@selector(setTrayTitle:), nsTitle);
}

void darwin_set_tooltip(const char *tooltip) {
    NSString *nsTooltip = (tooltip != NULL) ? [NSString stringWithUTF8String:tooltip] : @"";
    darwin_on_main(@selector(setTrayTooltip:), nsTooltip);
}

void darwin_show_popover(void) {
    darwin_on_main(@selector(showTrayPopover), nil);
}

void darwin_update_panel(const char *json) {
    if (json == NULL) {
        return;
    }
    NSString *jsonStr = [NSString stringWithUTF8String:json];
    darwin_on_main(@selector(updateTrayPanel:), jsonStr);
}

void darwin_quit(void) {
    darwin_on_main(@selector(terminateTray), nil);
}

void darwin_show_settings(const char *json) {
    if (json == NULL) {
        return;
    }
    NSString *jsonStr = [NSString stringWithUTF8String:json];
    darwin_on_main(@selector(showSettingsPanel:), jsonStr);
}

void darwin_hide_settings(void) {
    id target = [NSApp delegate];
    if (target == nil) {
        return;
    }
    [target performSelectorOnMainThread:@selector(hideSettingsPanel)
                           withObject:nil
                        waitUntilDone:YES];
}

void darwin_settings_show_error(const char *message) {
    NSString *msg = (message != NULL) ? [NSString stringWithUTF8String:message] : @"Unknown error";
    darwin_on_main(@selector(settingsSaveError:), msg);
}

void darwin_update_analyse_result(const char *json) {
    if (json == NULL) {
        return;
    }
    NSString *jsonStr = [NSString stringWithUTF8String:json];
    darwin_on_main(@selector(updateAnalyseResult:), jsonStr);
}
