// status_popover.m — NSStatusItem + NSPopover panel with Allow/Deny rows.
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-3.0-darwin-panel.md).

#import "status_popover.h"
#import <Cocoa/Cocoa.h>
#import <stdlib.h>

// Implemented in bindings.go via //export.
extern void goDecide(char *id, char *decision);
extern void goSetMode(int mode_index);
extern void goQuitTray(void);

static char *copyCString(NSString *s) {
    if (s == nil) {
        return NULL;
    }
    return strdup([s UTF8String]);
}

static const CGFloat kPanelWidth = 360.0;
static const CGFloat kPanelMinHeight = 220.0;
static const CGFloat kPanelMaxHeight = 480.0;
static const CGFloat kPadding = 12.0;
static const CGFloat kRowHeight = 36.0;

static NSStatusItem *gStatusItem = nil;
static id gTrayDelegate = nil;
static NSPopover *gPopover = nil;
static darwin_ready_fn gOnReady = NULL;

static NSTextField *gDaemonLabel = nil;
static NSTextField *gPendingLabel = nil;
static NSSegmentedControl *gModeControl = nil;
static NSStackView *gBodyStack = nil;
static NSScrollView *gBodyScroll = nil;
static NSView *gPanelRoot = nil;

static NSString *jsonString(id dict, NSString *key);
static BOOL jsonBool(id dict, NSString *key);
static NSInteger jsonInt(id dict, NSString *key);
static void rebuildBodyFromJSON(NSDictionary *payload, id target);

@interface VibeGuardTrayDelegate : NSObject <NSApplicationDelegate, NSPopoverDelegate>
@end

@implementation VibeGuardTrayDelegate

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
}

- (void)buildPanelShell {
    gPanelRoot = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, kPanelWidth, kPanelMinHeight)];

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

    gDaemonLabel = [self makeStatusLabel:@"● Daemon: …"];
    gPendingLabel = [self makeStatusLabel:@"● pending …"];
    [rootStack addArrangedSubview:gDaemonLabel];
    [rootStack addArrangedSubview:gPendingLabel];

    gModeControl = [NSSegmentedControl segmentedControlWithLabels:@[@"Ask", @"Auto-allow", @"Auto-deny"]
                                                         trackingMode:NSSegmentSwitchTrackingSelectOne
                                                               target:self
                                                               action:@selector(modeChanged:)];
    gModeControl.segmentDistribution = NSSegmentDistributionFillEqually;
    [rootStack addArrangedSubview:gModeControl];
    [gModeControl.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)].active = YES;

    gBodyStack = [[NSStackView alloc] init];
    gBodyStack.translatesAutoresizingMaskIntoConstraints = NO;
    gBodyStack.orientation = NSUserInterfaceLayoutOrientationVertical;
    gBodyStack.alignment = NSLayoutAttributeLeading;
    gBodyStack.spacing = 6.0;

    gBodyScroll = [[NSScrollView alloc] init];
    gBodyScroll.translatesAutoresizingMaskIntoConstraints = NO;
    gBodyScroll.hasVerticalScroller = YES;
    gBodyScroll.drawsBackground = NO;
    gBodyScroll.borderType = NSNoBorder;
    gBodyScroll.documentView = gBodyStack;
    NSView *clipView = gBodyScroll.contentView;
    [NSLayoutConstraint activateConstraints:@[
        [gBodyStack.leadingAnchor constraintEqualToAnchor:clipView.leadingAnchor],
        [gBodyStack.trailingAnchor constraintEqualToAnchor:clipView.trailingAnchor],
        [gBodyStack.topAnchor constraintEqualToAnchor:clipView.topAnchor],
        [gBodyStack.bottomAnchor constraintEqualToAnchor:clipView.bottomAnchor],
        [gBodyStack.widthAnchor constraintEqualToAnchor:clipView.widthAnchor],
    ]];
    [rootStack addArrangedSubview:gBodyScroll];
    [gBodyScroll.heightAnchor constraintGreaterThanOrEqualToConstant:80.0].active = YES;
    [gBodyScroll setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationVertical];

    NSButton *quitBtn = [NSButton buttonWithTitle:@"Quit" target:self action:@selector(quitClicked:)];
    quitBtn.bezelStyle = NSBezelStyleRounded;
    [rootStack addArrangedSubview:quitBtn];
    [quitBtn.centerXAnchor constraintEqualToAnchor:rootStack.centerXAnchor].active = YES;
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

- (void)modeChanged:(NSSegmentedControl *)sender {
    goSetMode((int)sender.selectedSegment);
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
    goQuitTray();
}

- (void)setTrayIcon:(NSImage *)image {
    if (gStatusItem == nil || image == nil) {
        return;
    }
    image.size = NSMakeSize(18.0, 18.0);
    [image setTemplate:YES];
    gStatusItem.button.image = image;
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
    if (gDaemonLabel == nil || gBodyStack == nil || jsonStr.length == 0) {
        return;
    }

    NSData *data = [jsonStr dataUsingEncoding:NSUTF8StringEncoding];
    NSError *err = nil;
    id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:&err];
    if (err != nil || ![parsed isKindOfClass:[NSDictionary class]]) {
        return;
    }
    NSDictionary *payload = (NSDictionary *)parsed;

    gDaemonLabel.stringValue = jsonString(payload, @"daemon_status");
    gPendingLabel.stringValue = jsonString(payload, @"pending_count");

    NSInteger modeIndex = jsonInt(payload, @"mode_index");
    if (modeIndex >= 0 && modeIndex < gModeControl.segmentCount) {
        gModeControl.selectedSegment = modeIndex;
    }
    gModeControl.enabled = jsonBool(payload, @"mode_enabled");

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

static NSView *makeApprovalRow(NSString *label, NSString *approvalID, id target) {
    NSView *row = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, kPanelWidth - (kPadding * 2), kRowHeight)];
    row.translatesAutoresizingMaskIntoConstraints = NO;

    NSTextField *labelField = [NSTextField labelWithString:label];
    labelField.font = [NSFont systemFontOfSize:[NSFont systemFontSize]];
    labelField.lineBreakMode = NSLineBreakByTruncatingTail;
    labelField.translatesAutoresizingMaskIntoConstraints = NO;
    [row addSubview:labelField];

    NSButton *allowBtn = [NSButton buttonWithTitle:@"Allow" target:target action:@selector(allowClicked:)];
    allowBtn.bezelStyle = NSBezelStyleRounded;
    allowBtn.identifier = approvalID;
    allowBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [row addSubview:allowBtn];

    NSButton *denyBtn = [NSButton buttonWithTitle:@"Deny" target:target action:@selector(denyClicked:)];
    denyBtn.bezelStyle = NSBezelStyleRounded;
    denyBtn.identifier = approvalID;
    denyBtn.translatesAutoresizingMaskIntoConstraints = NO;
    [row addSubview:denyBtn];

    [NSLayoutConstraint activateConstraints:@[
        [labelField.leadingAnchor constraintEqualToAnchor:row.leadingAnchor],
        [labelField.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [labelField.trailingAnchor constraintLessThanOrEqualToAnchor:allowBtn.leadingAnchor constant:-8.0],

        [denyBtn.trailingAnchor constraintEqualToAnchor:row.trailingAnchor],
        [denyBtn.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [denyBtn.widthAnchor constraintGreaterThanOrEqualToConstant:52.0],

        [allowBtn.trailingAnchor constraintEqualToAnchor:denyBtn.leadingAnchor constant:-6.0],
        [allowBtn.centerYAnchor constraintEqualToAnchor:row.centerYAnchor],
        [allowBtn.widthAnchor constraintGreaterThanOrEqualToConstant:52.0],

        [row.heightAnchor constraintEqualToConstant:kRowHeight],
        [row.widthAnchor constraintEqualToConstant:kPanelWidth - (kPadding * 2)],
    ]];

    return row;
}

static void rebuildBodyFromJSON(NSDictionary *payload, id target) {
    for (NSView *subview in [gBodyStack.arrangedSubviews copy]) {
        [gBodyStack removeArrangedSubview:subview];
        [subview removeFromSuperview];
    }

    NSArray *rows = payload[@"rows"];
    if ([rows isKindOfClass:[NSArray class]]) {
        for (id rowObj in rows) {
            if (![rowObj isKindOfClass:[NSDictionary class]]) {
                continue;
            }
            NSDictionary *row = (NSDictionary *)rowObj;
            NSString *rowID = jsonString(row, @"id");
            NSString *rowLabel = jsonString(row, @"label");
            if (rowID.length == 0) {
                continue;
            }
            NSView *rowView = makeApprovalRow(rowLabel, rowID, target);
            [gBodyStack addArrangedSubview:rowView];
        }
    }

    NSString *overflow = jsonString(payload, @"overflow_hint");
    if (overflow.length > 0) {
        NSTextField *hint = [NSTextField labelWithString:overflow];
        hint.font = [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
        hint.textColor = [NSColor secondaryLabelColor];
        [gBodyStack addArrangedSubview:hint];
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
    CGFloat bodyHeight = NSHeight(gBodyStack.frame);
    CGFloat clamped = MIN(MAX(bodyHeight, 80.0), kPanelMaxHeight - 160.0);
    for (NSLayoutConstraint *c in gBodyScroll.constraints) {
        if (c.firstAttribute == NSLayoutAttributeHeight && c.firstItem == gBodyScroll) {
            c.constant = clamped;
            break;
        }
    }
}

void darwin_tray_prepare(darwin_ready_fn on_ready) {
    @autoreleasepool {
        gOnReady = on_ready;

        [NSApplication sharedApplication];
        gTrayDelegate = [[VibeGuardTrayDelegate alloc] init];
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
