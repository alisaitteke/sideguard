// status_popover.m — NSStatusItem + NSPopover panel with pending/history rows.
// See docs/plans/2026-07-02-1226-tray-ui-polish/ (tup-phase-3.0-tray-darwin.md).

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
static const CGFloat kPanelHeight = 480.0;
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
static NSTextField *gFooterDaemonLabel = nil;
static NSTextField *gFooterPendingLabel = nil;
static NSStackView *gUpdateFooter = nil;
static NSTextField *gUpdateLabel = nil;
static NSButton *gInstallBtn = nil;

static BOOL gHistoryHasMore = NO;
static NSView *gBodyContainer = nil;
static NSView *gDetailContainer = nil;
static NSScrollView *gDetailScroll = nil;
static NSTextView *gDetailTextView = nil;
static BOOL gShowingDetail = NO;
static NSString *gDetailRowID = nil;
static BOOL gDetailUseEventID = NO;
static NSButton *gAnalyseBtn = nil;
static NSStackView *gDetailActionStack = nil;
static NSButton *gDetailRunBtn = nil;
static NSButton *gDetailDeclineBtn = nil;
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
static char kRowDetailTextKey;
static char kRowDetailIDKey;
static char kRowDetailUseEventIDKey;

static NSString *jsonString(id dict, NSString *key);
static BOOL jsonBool(id dict, NSString *key);
static NSInteger jsonInt(id dict, NSString *key);
static void rebuildBodyFromJSON(NSDictionary *payload, id target);
static void addRowsFromJSONArray(NSArray *rows, NSString *kind, id target, NSMutableArray *outViews);
static NSView *makeHistoryRow(NSString *label, NSString *detail, NSString *rowID, id target);
static NSView *makeApprovalRow(NSString *label, NSString *detail, NSString *approvalID, id target);
static NSView *makeSectionLabel(NSString *text);
static NSView *makeLoadMoreRow(id target);
static void attachRowDetailTap(NSView *view, NSString *detail, NSString *rowID, BOOL useEventID, id target);
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
static void updateDetailActionButtons(NSString *rowID, BOOL useEventID, NSDictionary *payload);
static NSString *verdictDisplayTitle(NSString *verdict);
static NSColor *verdictBadgeColor(NSString *verdict);

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
    gPanelRoot = [[SideGuardPanelRootView alloc] initWithFrame:NSMakeRect(0, 0, kPanelWidth, kPanelHeight)];

    [NSLayoutConstraint activateConstraints:@[
        [gPanelRoot.widthAnchor constraintEqualToConstant:kPanelWidth],
        [gPanelRoot.heightAnchor constraintEqualToConstant:kPanelHeight],
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
    [NSLayoutConstraint activateConstraints:@[
        [gBodyScroll.topAnchor constraintEqualToAnchor:gBodyContainer.topAnchor],
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

    gDetailContainer = [[NSView alloc] init];
    gDetailContainer.translatesAutoresizingMaskIntoConstraints = NO;
    gDetailContainer.hidden = YES;
    [gBodyContainer addSubview:gDetailContainer];
    [NSLayoutConstraint activateConstraints:@[
        [gDetailContainer.topAnchor constraintEqualToAnchor:gBodyContainer.topAnchor],
        [gDetailContainer.bottomAnchor constraintEqualToAnchor:gBodyContainer.bottomAnchor],
        [gDetailContainer.leadingAnchor constraintEqualToAnchor:gBodyContainer.leadingAnchor],
        [gDetailContainer.trailingAnchor constraintEqualToAnchor:gBodyContainer.trailingAnchor],
    ]];

    NSStackView *detailStack = [[NSStackView alloc] init];
    detailStack.translatesAutoresizingMaskIntoConstraints = NO;
    detailStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    detailStack.alignment = NSLayoutAttributeLeading;
    detailStack.spacing = 8.0;
    [gDetailContainer addSubview:detailStack];
    [NSLayoutConstraint activateConstraints:@[
        [detailStack.topAnchor constraintEqualToAnchor:gDetailContainer.topAnchor],
        [detailStack.bottomAnchor constraintEqualToAnchor:gDetailContainer.bottomAnchor],
        [detailStack.leadingAnchor constraintEqualToAnchor:gDetailContainer.leadingAnchor],
        [detailStack.trailingAnchor constraintEqualToAnchor:gDetailContainer.trailingAnchor],
    ]];

    NSButton *backBtn = [NSButton buttonWithTitle:@"← History" target:self action:@selector(detailBackClicked:)];
    backBtn.bezelStyle = NSBezelStyleRounded;
    backBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [detailStack addArrangedSubview:backBtn];

    gDetailScroll = [[NSScrollView alloc] init];
    gDetailScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gDetailScroll.hasVerticalScroller = YES;
    gDetailScroll.autohidesScrollers = YES;
    gDetailScroll.drawsBackground = NO;
    gDetailScroll.borderType = NSNoBorder;

    gDetailTextView = [[NSTextView alloc] init];
    gDetailTextView.editable = NO;
    gDetailTextView.selectable = YES;
    gDetailTextView.drawsBackground = NO;
    gDetailTextView.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    gDetailTextView.textColor = [NSColor labelColor];
    gDetailTextView.textContainerInset = NSMakeSize(4.0, 4.0);
    gDetailTextView.textContainer.widthTracksTextView = YES;
    gDetailTextView.textContainer.containerSize = NSMakeSize(0, CGFLOAT_MAX);
    gDetailTextView.verticallyResizable = YES;
    gDetailTextView.horizontallyResizable = NO;
    gDetailTextView.autoresizingMask = NSViewWidthSizable;
    gDetailScroll.documentView = gDetailTextView;
    [detailStack addArrangedSubview:gDetailScroll];
    [gDetailScroll setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationVertical];
    [gDetailScroll setContentCompressionResistancePriority:NSLayoutPriorityDefaultLow
                                          forOrientation:NSLayoutConstraintOrientationVertical];

    [gDetailScroll setContentCompressionResistancePriority:NSLayoutPriorityDefaultLow
                                          forOrientation:NSLayoutConstraintOrientationVertical];

    gAnalyseBtn = [NSButton buttonWithTitle:@"Analyse" target:self action:@selector(analyseClicked:)];
    gAnalyseBtn.bezelStyle = NSBezelStyleRounded;
    gAnalyseBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [detailStack addArrangedSubview:gAnalyseBtn];

    gAnalyseStatusLabel = [NSTextField labelWithString:@""];
    gAnalyseStatusLabel.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    gAnalyseStatusLabel.textColor = [NSColor secondaryLabelColor];
    gAnalyseStatusLabel.hidden = YES;
    [detailStack addArrangedSubview:gAnalyseStatusLabel];

    gVerdictBadge = [NSTextField labelWithString:@""];
    gVerdictBadge.font = [NSFont systemFontOfSize:[NSFont systemFontSize] weight:NSFontWeightSemibold];
    gVerdictBadge.hidden = YES;
    [detailStack addArrangedSubview:gVerdictBadge];

    gAnalyseSummaryLabel = [NSTextField labelWithString:@""];
    gAnalyseSummaryLabel.font = [NSFont systemFontOfSize:[NSFont systemFontSize] weight:NSFontWeightMedium];
    gAnalyseSummaryLabel.textColor = [NSColor labelColor];
    gAnalyseSummaryLabel.hidden = YES;
    [detailStack addArrangedSubview:gAnalyseSummaryLabel];

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
    [detailStack addArrangedSubview:gAnalyseExplanationScroll];
    [gAnalyseExplanationScroll.heightAnchor constraintLessThanOrEqualToConstant:96.0].active = YES;
    [gAnalyseExplanationScroll setContentHuggingPriority:NSLayoutPriorityRequired
                                        forOrientation:NSLayoutConstraintOrientationVertical];

    NSView *detailActionsSpacer = [[NSView alloc] init];
    [detailActionsSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow
                                    forOrientation:NSLayoutConstraintOrientationVertical];
    [detailStack addArrangedSubview:detailActionsSpacer];

    gDetailActionStack = [[NSStackView alloc] init];
    gDetailActionStack.translatesAutoresizingMaskIntoConstraints = NO;
    gDetailActionStack.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    gDetailActionStack.alignment = NSLayoutAttributeCenterY;
    gDetailActionStack.spacing = 8.0;
    gDetailActionStack.hidden = YES;

    gDetailRunBtn = [NSButton buttonWithTitle:@"Run" target:self action:@selector(allowClicked:)];
    gDetailRunBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(gDetailRunBtn, @"Run", [NSColor systemGreenColor]);
    [gDetailActionStack addArrangedSubview:gDetailRunBtn];

    NSView *detailActionSpacer = [[NSView alloc] init];
    [detailActionSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow
                                   forOrientation:NSLayoutConstraintOrientationHorizontal];
    [gDetailActionStack addArrangedSubview:detailActionSpacer];

    gDetailDeclineBtn = [NSButton buttonWithTitle:@"Decline" target:self action:@selector(denyClicked:)];
    gDetailDeclineBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(gDetailDeclineBtn, @"Decline", [NSColor systemRedColor]);
    [gDetailActionStack addArrangedSubview:gDetailDeclineBtn];
    [detailStack addArrangedSubview:gDetailActionStack];
    [gDetailActionStack.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;
    [gDetailRunBtn.widthAnchor constraintGreaterThanOrEqualToConstant:72.0].active = YES;
    [gDetailDeclineBtn.widthAnchor constraintGreaterThanOrEqualToConstant:80.0].active = YES;

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

    // Footer: daemon + pending status.
    NSStackView *footerStatusStack = [[NSStackView alloc] init];
    footerStatusStack.translatesAutoresizingMaskIntoConstraints = NO;
    footerStatusStack.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    footerStatusStack.alignment = NSLayoutAttributeCenterY;
    footerStatusStack.spacing = 12.0;

    gFooterDaemonLabel = [self makeStatusLabel:@"● Daemon: …"];
    gFooterPendingLabel = [self makeStatusLabel:@"● pending …"];
    [footerStatusStack addArrangedSubview:gFooterDaemonLabel];
    [footerStatusStack addArrangedSubview:gFooterPendingLabel];
    [rootStack addArrangedSubview:footerStatusStack];
    [footerStatusStack.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;

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

- (void)showCommandDetail:(NSString *)detail rowID:(NSString *)rowID useEventID:(BOOL)useEventID {
    if (gDetailTextView == nil || gBodyScroll == nil || gDetailContainer == nil) {
        return;
    }
    clearAnalyseUI();
    gDetailRowID = rowID.length > 0 ? [rowID copy] : nil;
    gDetailUseEventID = useEventID;
    gDetailTextView.string = detail ?: @"";
    [gDetailTextView scrollPoint:NSMakePoint(0, 0)];
    updateDetailActionButtons(rowID, useEventID, nil);
    gBodyScroll.hidden = YES;
    gDetailContainer.hidden = NO;
    gShowingDetail = YES;
}

- (void)detailBackClicked:(id)sender {
    (void)sender;
    if (gBodyScroll == nil || gDetailContainer == nil) {
        return;
    }
    clearAnalyseUI();
    gBodyScroll.hidden = NO;
    gDetailContainer.hidden = YES;
    gShowingDetail = NO;
    gDetailRowID = nil;
    gDetailUseEventID = NO;
    if (gDetailActionStack != nil) {
        gDetailActionStack.hidden = YES;
    }
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

    NSString *rowID = gDetailRowID ?: @"";
    NSString *command = gDetailTextView.string ?: @"";
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
    goAnalyseCommand(rowCopy, cmdCopy, gDetailUseEventID ? 1 : 0);
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

    gBodyScroll.hidden = YES;
    gDetailContainer.hidden = YES;
    gSettingsContainer.hidden = NO;
    gShowingDetail = NO;
    gShowingSettings = YES;
    gDetailRowID = nil;
    gDetailUseEventID = NO;
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
    if (view == nil) {
        return;
    }
    NSString *detail = objc_getAssociatedObject(view, &kRowDetailTextKey);
    if (detail.length == 0) {
        return;
    }
    NSString *rowID = objc_getAssociatedObject(view, &kRowDetailIDKey);
    NSNumber *useEventNum = objc_getAssociatedObject(view, &kRowDetailUseEventIDKey);
    BOOL useEventID = useEventNum != nil ? useEventNum.boolValue : NO;
    [self showCommandDetail:detail rowID:rowID useEventID:useEventID];
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

    gHistoryHasMore = jsonBool(payload, @"history_has_more");

    gFooterDaemonLabel.stringValue = jsonString(payload, @"footer_daemon");
    gFooterPendingLabel.stringValue = jsonString(payload, @"footer_pending");

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
    if (gDetailTextView != nil) {
        gDetailTextView.textColor = [NSColor labelColor];
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

static BOOL rowIDIsPending(NSString *rowID, NSDictionary *payload) {
    if (rowID.length == 0 || payload == nil) {
        return NO;
    }
    NSArray *pendingRows = payload[@"pending_rows"];
    if (![pendingRows isKindOfClass:[NSArray class]]) {
        return NO;
    }
    for (id rowObj in pendingRows) {
        if (![rowObj isKindOfClass:[NSDictionary class]]) {
            continue;
        }
        if ([rowID isEqualToString:jsonString((NSDictionary *)rowObj, @"id")]) {
            return YES;
        }
    }
    return NO;
}

static void updateDetailActionButtons(NSString *rowID, BOOL useEventID, NSDictionary *payload) {
    BOOL actionable = !useEventID && rowID.length > 0;
    if (actionable && payload != nil) {
        actionable = rowIDIsPending(rowID, payload);
    }
    if (gDetailActionStack != nil) {
        gDetailActionStack.hidden = !actionable;
    }
    if (actionable) {
        if (gDetailRunBtn != nil) {
            gDetailRunBtn.identifier = rowID;
        }
        if (gDetailDeclineBtn != nil) {
            gDetailDeclineBtn.identifier = rowID;
        }
    }
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

static NSView *makeSectionLabel(NSString *text) {
    NSTextField *field = [NSTextField labelWithString:text];
    field.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize] weight:NSFontWeightSemibold];
    field.textColor = [NSColor tertiaryLabelColor];
    return field;
}

static void attachRowDetailTap(NSView *view, NSString *detail, NSString *rowID, BOOL useEventID, id target) {
    if (view == nil) {
        return;
    }
    NSString *text = detail.length > 0 ? detail : @"";
    objc_setAssociatedObject(view, &kRowDetailTextKey, text, OBJC_ASSOCIATION_COPY_NONATOMIC);
    NSString *idText = rowID.length > 0 ? rowID : @"";
    objc_setAssociatedObject(view, &kRowDetailIDKey, idText, OBJC_ASSOCIATION_COPY_NONATOMIC);
    objc_setAssociatedObject(view, &kRowDetailUseEventIDKey, @(useEventID), OBJC_ASSOCIATION_RETAIN_NONATOMIC);

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

static NSView *makeHistoryRow(NSString *label, NSString *detail, NSString *rowID, id target) {
    NSString *fullDetail = detail.length > 0 ? detail : label;
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

    attachRowDetailTap(row, fullDetail, rowID, YES, target);
    return row;
}

static NSView *makeApprovalRow(NSString *label, NSString *detail, NSString *approvalID, id target) {
    NSString *fullDetail = detail.length > 0 ? detail : label;
    CGFloat rowWidth = kPanelWidth - (kPadding * 2);

    NSView *row = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, rowWidth, kRowHeight)];
    row.translatesAutoresizingMaskIntoConstraints = NO;
    row.wantsLayer = YES;
    row.layer.backgroundColor = [[NSColor systemOrangeColor] colorWithAlphaComponent:0.18].CGColor;
    row.layer.cornerRadius = 6.0;

    NSTextField *labelField = [NSTextField labelWithString:label];
    labelField.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    labelField.textColor = [NSColor labelColor];
    labelField.lineBreakMode = NSLineBreakByTruncatingTail;
    labelField.translatesAutoresizingMaskIntoConstraints = NO;

    NSView *labelTapArea = [[NSView alloc] init];
    labelTapArea.translatesAutoresizingMaskIntoConstraints = NO;
    [labelTapArea addSubview:labelField];
    [row addSubview:labelTapArea];

    NSButton *allowBtn = [NSButton buttonWithTitle:@"Run" target:target action:@selector(allowClicked:)];
    allowBtn.identifier = approvalID;
    allowBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(allowBtn, @"Run", [NSColor systemGreenColor]);
    [row addSubview:allowBtn];

    NSButton *denyBtn = [NSButton buttonWithTitle:@"Decline" target:target action:@selector(denyClicked:)];
    denyBtn.identifier = approvalID;
    denyBtn.translatesAutoresizingMaskIntoConstraints = NO;
    styleColoredActionButton(denyBtn, @"Decline", [NSColor systemRedColor]);
    [row addSubview:denyBtn];

    [NSLayoutConstraint activateConstraints:@[
        [row.widthAnchor constraintEqualToConstant:rowWidth],
        [labelTapArea.leadingAnchor constraintEqualToAnchor:row.leadingAnchor],
        [labelTapArea.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [labelTapArea.trailingAnchor constraintEqualToAnchor:allowBtn.leadingAnchor constant:-8.0],
        [labelTapArea.heightAnchor constraintEqualToAnchor:row.heightAnchor],

        [labelField.leadingAnchor constraintEqualToAnchor:labelTapArea.leadingAnchor],
        [labelField.trailingAnchor constraintEqualToAnchor:labelTapArea.trailingAnchor],
        [labelField.centerYAnchor constraintEqualToAnchor:labelTapArea.centerYAnchor],

        [denyBtn.trailingAnchor constraintEqualToAnchor:row.trailingAnchor],
        [denyBtn.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [denyBtn.widthAnchor constraintGreaterThanOrEqualToConstant:72.0],

        [allowBtn.trailingAnchor constraintEqualToAnchor:denyBtn.leadingAnchor constant:-6.0],
        [allowBtn.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [allowBtn.widthAnchor constraintGreaterThanOrEqualToConstant:56.0],

        [row.heightAnchor constraintEqualToConstant:kRowHeight],
    ]];

    attachRowDetailTap(labelTapArea, fullDetail, approvalID, NO, target);
    return row;
}

static void addRowsFromJSONArray(NSArray *rows, NSString *kind, id target, NSMutableArray *outViews) {
    if (![rows isKindOfClass:[NSArray class]]) {
        return;
    }
    for (id rowObj in rows) {
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
        if ([kind isEqualToString:@"history"]) {
            [outViews addObject:makeHistoryRow(rowLabel, rowDetail, rowID, target)];
        } else {
            [outViews addObject:makeApprovalRow(rowLabel, rowDetail, rowID, target)];
        }
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
    if (gBodyScroll == nil || gSettingsContainer == nil) {
        return;
    }
    gBodyScroll.hidden = NO;
    gSettingsContainer.hidden = YES;
    gShowingSettings = NO;
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

static void rebuildBodyFromJSON(NSDictionary *payload, id target) {
    BOOL wasShowingDetail = gShowingDetail;
    BOOL wasShowingSettings = gShowingSettings;
    NSString *detailRowID = wasShowingDetail ? [gDetailRowID copy] : nil;
    BOOL detailUseEventID = wasShowingDetail ? gDetailUseEventID : NO;

    NSClipView *clipView = gBodyScroll ? gBodyScroll.contentView : nil;
    NSPoint savedOrigin = NSZeroPoint;
    CGFloat docHeightBefore = 0;
    BOOL preserveScroll = clipView != nil && !wasShowingDetail;
    if (preserveScroll) {
        savedOrigin = clipView.bounds.origin;
        if (clipView.documentView != nil) {
            docHeightBefore = NSHeight(clipView.documentView.bounds);
        }
    }

    for (NSView *subview in [gBodyStack.arrangedSubviews copy]) {
        [gBodyStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }

    NSMutableArray *pendingViews = [NSMutableArray array];
    addRowsFromJSONArray(payload[@"pending_rows"], @"pending", target, pendingViews);
    for (NSView *view in pendingViews) {
        [gBodyStack addArrangedSubview:view];
    }

    NSString *overflow = jsonString(payload, @"pending_overflow");
    if (overflow.length > 0) {
        NSTextField *hint = [NSTextField labelWithString:overflow];
        hint.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
        hint.textColor = [NSColor secondaryLabelColor];
        [gBodyStack addArrangedSubview:hint];
    }

    NSArray *historyRows = payload[@"history_rows"];
    BOOL hasHistory = [historyRows isKindOfClass:[NSArray class]] && historyRows.count > 0;
    if (hasHistory) {
        [gBodyStack addArrangedSubview:makeSectionLabel(@"History")];
        NSMutableArray *historyViews = [NSMutableArray array];
        addRowsFromJSONArray(historyRows, @"history", target, historyViews);
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

    [gBodyStack layoutSubtreeIfNeeded];

    if (preserveScroll && clipView.documentView != nil) {
        CGFloat docHeightAfter = NSHeight(clipView.documentView.bounds);
        CGFloat heightDelta = docHeightAfter - docHeightBefore;
        // Pending rows prepend at the top; compensate scroll when the user had moved down.
        if (heightDelta > 0 && savedOrigin.y > 0) {
            savedOrigin.y += heightDelta;
        }
        NSRect docBounds = clipView.documentView.bounds;
        CGFloat maxY = MAX(0, NSMaxY(docBounds) - NSHeight(clipView.bounds));
        savedOrigin.y = MIN(MAX(0, savedOrigin.y), maxY);
        [clipView scrollToPoint:savedOrigin];
        [gBodyScroll reflectScrolledClipView:clipView];
    }

    if (wasShowingDetail) {
        gBodyScroll.hidden = YES;
        gDetailContainer.hidden = NO;
        gShowingDetail = YES;
        gDetailRowID = detailRowID;
        gDetailUseEventID = detailUseEventID;
        if (gDetailTextView != nil && detailRowID.length > 0) {
            NSString *updated = detailTextForRowID(payload, detailRowID);
            if (updated.length > 0) {
                gDetailTextView.string = updated;
            }
        }
        updateDetailActionButtons(detailRowID, detailUseEventID, payload);
    } else if (wasShowingSettings) {
        gBodyScroll.hidden = YES;
        gDetailContainer.hidden = YES;
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
