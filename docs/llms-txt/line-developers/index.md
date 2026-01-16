# LINE Developers

> Official documentation for the LINE Platform, including Messaging API, LINE Login, LIFF, LINE MINI App, and other LINE Platform services provided by LY Corporation.

## Docs

- [LINE Platform basics](basics/): Core concepts of the LINE Platform
- [LINE Developers Console](line-developers-console/): Manage channels, roles, and settings
- [Messaging API](messaging-api/): Build bots and send messages
- [LINE Login](line-login/): OAuth 2.0-based authentication
- [LIFF](liff/): LINE Front-end Framework for web apps
- [LINE MINI App](line-mini-app/): Web apps running inside LINE
- [Partner Docs](partner-docs/): Options for corporate customers

## LINE Platform basics

- [Channel access token](basics/channel-access-token.md): Issue and use channel access tokens for LINE API authentication
- [Get user profile information](basics/user-profile.md): Retrieve user profile information using the LINE API
- [LINE API Status](basics/line-api-status.md): Check availability of the LINE Platform

## LINE Developers Console

- [Overview](line-developers-console/overview.md): Overview of the LINE Developers Console
- [Log in to LINE Developers](line-developers-console/login-account.md): How to log in and access the Console
- [Managing roles](line-developers-console/managing-roles.md): Manage user roles and permissions
- [Best practices](line-developers-console/best-practices-for-provider-and-channel-management.md): Provider and channel management best practices
- [Notifications](line-developers-console/notification.md): Set up email or notification center alerts

## Messaging API

- [Overview](messaging-api/overview.md): Overview of the Messaging API
- [Getting started](messaging-api/getting-started.md): Step-by-step guide for getting started
- [Development guidelines](messaging-api/development-guidelines.md): Rate limits and error handling
- [Message types](messaging-api/message-types.md): Various message formats supported
- [Build a bot](messaging-api/building-bot.md): Build a bot with webhooks
- [Pricing](messaging-api/pricing.md): Free and paid plans, message quotas
- [Send messages](messaging-api/sending-messages.md): How to send messages
- [Character counting](messaging-api/text-character-count.md): Character counting in text messages
- [Get user IDs](messaging-api/getting-user-ids.md): Retrieve user ID or group ID
- [Stickers](messaging-api/stickers.md): Sticker and package IDs catalog
- [Using audiences](messaging-api/using-audience.md): Create and target audiences
- [Using quick replies](messaging-api/using-quick-reply.md): Implement quick replies
- [Statistics](messaging-api/unit-based-statistics-aggregation.md): Access usage statistics
- [LINE URL scheme](messaging-api/using-line-url-scheme.md): Launch LINE features via URL schemes
- [Using beacons](messaging-api/using-beacons.md): Trigger messages based on proximity
- [Gain friends](messaging-api/sharing-bot.md): Increase friends via share links, QR codes
- [Account linking](messaging-api/linking-accounts.md): Link LINE accounts with external services
- [Icon and display name](messaging-api/icon-nickname-switch.md): Customize bot's icon and display name
- [Loading animation](messaging-api/use-loading-indicator.md): Show loading indicator
- [Membership features](messaging-api/use-membership-features.md): Implement membership levels
- [Send coupons](messaging-api/send-coupons-to-users.md): Create and send coupons
- [Quote tokens](messaging-api/get-quote-tokens.md): Reply to specific messages
- [Mark as read](messaging-api/mark-as-read.md): Mark messages as read programmatically
- [Retry API requests](messaging-api/retrying-api-request.md): Retry strategies and idempotency
- [Stop using Official Account](messaging-api/stop-using-line-official-account.md): Deactivate or delete your account
- [Stop using Messaging API](messaging-api/stop-using-messaging-api.md): Shut down Messaging API usage
- [Measure impressions](messaging-api/measure-impressions.md): How impressions are measured
- [Node.js tutorial](messaging-api/nodejs-sample.md): Build a simple LINE bot with Node.js
- [Receive messages (webhook)](messaging-api/receiving-messages.md): Receive messages via webhooks
- [Verify webhook URL](messaging-api/verify-webhook-url.md): Steps to verify your webhook endpoint
- [Verify webhook signature](messaging-api/verify-webhook-signature.md): Validate signatures with HMAC-SHA256
- [Webhook error statistics](messaging-api/check-webhook-error-statistics.md): Monitor webhook errors
- [SSL/TLS specification](messaging-api/ssl-tls-spec-of-the-webhook-source.md): SSL/TLS requirements for webhooks
- [Rich menus overview](messaging-api/rich-menus-overview.md): Introduction to rich menus
- [Use rich menus](messaging-api/using-rich-menus.md): Configure and link rich menus
- [Per-user rich menus](messaging-api/use-per-user-rich-menus.md): Personalized rich menus
- [Switch rich menu tabs](messaging-api/switch-rich-menus.md): Multi-section navigation
- [Play with rich menus](messaging-api/try-rich-menu.md): Preview and test rich menus
- [LINE Bot Designer](messaging-api/using-bot-designer.md): Visual bot design interface
- [Download Bot Designer](messaging-api/download-bot-designer.md): Download instructions
- [Send Flex Messages](messaging-api/using-flex-messages.md): Rich, customizable layouts
- [Flex Message elements](messaging-api/flex-message-elements.md): UI components (box, text, image, button)
- [Flex Message layout](messaging-api/flex-message-layout.md): Layout rules and best practices
- [Flex Message with video](messaging-api/create-flex-message-including-video.md): Embed videos in Flex Messages
- [Flex Message Simulator tutorial](messaging-api/using-flex-message-simulator.md): Test Flex Messages with simulator
- [Actions](messaging-api/actions.md): Postback, URI, message, and camera actions
- [Group chats](messaging-api/group-chats.md): Handle group and multi-person chat events
- [User consent](messaging-api/user-consent.md): Obtain consent for profile data
- [Beacon device spec](messaging-api/beacon-device-spec.md): BLE beacon specifications
- [Secure message sample](messaging-api/secure-message-sample.md): Encrypted message examples
- [LINE Bot SDKs](messaging-api/line-bot-sdk.md): Official SDKs (Node.js, Java, Python, etc.)
- [Issue channel access token v2.1](messaging-api/generate-json-web-token.md): Sign JWT and issue tokens

## LINE Social Plugins

- [Share buttons](line-social-plugins/share-buttons.md): Integrate Share buttons

## LINE Login

- [Overview](line-login/overview.md): LINE Login features summary
- [Getting started](line-login/getting-started.md): Set up and test LINE Login
- [Development guidelines](line-login/development-guidelines.md): Building secure integrations
- [Security checklist](line-login/security-checklist.md): Security best practices
- [Integrate with web app](line-login/integrate-line-login.md): OAuth 2.0 flow and SDK tools
- [Handle auto login failure](line-login/how-to-handle-auto-login-failure.md): Resolve auto login issues
- [PKCE support](line-login/integrate-pkce.md): Implement PKCE for enhanced security
- [Add friend option](line-login/link-a-bot.md): Prompt users to add Official Account
- [Secure login process](line-login/secure-login-process.md): Exchange ID tokens securely
- [Managing access tokens](line-login/managing-access-tokens.md): Verify, refresh, and revoke tokens
- [Verify ID token](line-login/verify-id-token.md): Decode and validate ID tokens
- [Managing users](line-login/managing-users.md): Logout and access revocation
- [Managing authorized apps](line-login/managing-authorized-apps.md): View or revoke app authorizations
- [Login button](line-login/login-button.md): Design and implementation guide
- [LINE URL scheme](line-login/using-line-url-scheme.md): Launch LINE actions via URL schemes

## LINE Login SDKs

- [iOS SDK overview](line-login-sdks/ios-sdk-swift-overview.md): LINE SDK for iOS (Swift) overview
- [Android SDK overview](line-login-sdks/android-sdk-overview.md): LINE SDK for Android overview

## LIFF (LINE Front-end Framework)

- [Overview](liff/overview.md): Introduction to LIFF
- [Getting started](liff/getting-started.md): Set up LINE Login channel for LIFF
- [Development guidelines](liff/development-guidelines.md): Best practices for LIFF apps
- [Trying the starter app](liff/trying-liff-app.md): Test prebuilt LIFF app
- [Create LIFF App CLI](liff/cli-tool-create-liff-app.md): Scaffold a LIFF project
- [Developing LIFF apps](liff/developing-liff-apps.md): Implement front-end functionality
- [Registering LIFF apps](liff/registering-liff-apps.md): Link app to LINE Login channel
- [Opening LIFF app](liff/opening-liff-app.md): Launch via URL or QR code
- [Minimizing LIFF browser](liff/minimizing-liff-browser.md): Minimize browser window
- [Using user profile](liff/using-user-profile.md): Access LINE user profile data
- [LIFF vs LINE in-app browser](liff/differences-between-liff-browser-and-line-in-app-browser.md): Key differences
- [LIFF vs external browser](liff/differences-between-liff-browser-and-external-browser.md): Key differences
- [LIFF plugin](liff/liff-plugin.md): Extend apps with plugins
- [Pluggable SDK](liff/pluggable-sdk.md): Choose which LIFF APIs to include
- [LIFF CLI](liff/liff-cli.md): Build, test, and deploy LIFF apps
- [Versioning policy](liff/versioning-policy.md): Version strategy and compatibility
- [Release notes](liff/release-notes.md): Updates and new features

## LINE MINI App

- [Quickstart](line-mini-app/quickstart.md): Create and deploy a LINE MINI App
- [Development guidelines](line-mini-app/development-guidelines.md): Technical guidance

### Discover

- [Introduction](line-mini-app/discover/introduction.md): Overview of LINE MINI Apps
- [Console guide](line-mini-app/discover/console-guide.md): Manage via LINE Developers Console
- [Specifications](line-mini-app/discover/specifications.md): Technical specifications
- [Built-in features](line-mini-app/discover/builtin-features.md): Native features available
- [Custom features](line-mini-app/discover/custom-features.md): Extend with custom features
- [UI components](line-mini-app/discover/ui-components.md): Available UI components

### Design

- [App icon](line-mini-app/design/line-mini-app-icon.md): Icon design guidelines
- [Safe area](line-mini-app/design/landscape.md): Layout within safe display areas
- [Loading icon](line-mini-app/design/loading-icon.md): Loading indicator guidelines

### Develop

- [Getting started](line-mini-app/develop/develop-overview.md): Development flow and setup
- [Share messages](line-mini-app/develop/share-messages.md): Custom action buttons
- [Service messages](line-mini-app/develop/service-messages.md): Send service messages
- [Custom path](line-mini-app/develop/custom-path.md): Customize LIFF URL
- [Skip consent](line-mini-app/develop/channel-consent-simplification.md): Streamline consent screens
- [Payments](line-mini-app/develop/payment.md): Implement payment features
- [Permanent links](line-mini-app/develop/permanent-links.md): Create permanent URLs
- [Add to home screen](line-mini-app/develop/add-to-home-screen.md): Pin to device home screen
- [Console settings](line-mini-app/develop/configure-console.md): Configure app settings
- [External browser](line-mini-app/develop/external-browser.md): Launch outside LINE app
- [Web to MINI App](line-mini-app/develop/web-to-mini-app.md): Adapt existing web apps
- [Performance guidelines](line-mini-app/develop/performance-guidelines.md): Optimization tips

### Quick-fill

- [Overview](line-mini-app/quick-fill/overview.md): Auto-filling forms
- [Design regulations](line-mini-app/quick-fill/design-regulations.md): UI/UX guidelines

### Submit

- [Submission guide](line-mini-app/submit/submission-guide.md): Submit and publish your app

### Service

- [Service operation](line-mini-app/service/service-operation.md): Maintain after launch
- [LINE MINI App ads](line-mini-app/service/line-mini-app-ads.md): Monetize with Yahoo! JAPAN Ads
- [Update verified app](line-mini-app/service/update-service.md): Re-review after updates
- [Using Official Account](line-mini-app/service/line-mini-app-oa.md): Enhance with Official Account

## Partner Docs (Corporate Customers)

- [Overview](partner-docs/overview.md): Options for corporate customers API
- [Development guidelines](partner-docs/development-guidelines.md): Best practices for corporate API
- [API Policy Handbook](partner-docs/api-policy-handbook.md): Usage policies and restrictions
- [Notice](partner-docs/notice.md): Important notices for corporate users
- [Error notification](partner-docs/error-notification.md): Error codes and delivery failures
- [Provider page](partner-docs/provider-page.md): Manage provider page
- [Mission Stickers](partner-docs/mission-stickers.md): Mission Stickers API
- [LINE Profile+](partner-docs/line-profile-plus.md): Custom user attributes
- [LINE Beacon](partner-docs/line-beacon.md): User-side beacon requirements
- [Mark as read (old)](partner-docs/mark-as-read.md): Old Mark as Read API
- [Module](partner-docs/module.md): Dynamic chatbot control
- [Attach module channel](partner-docs/module-technical-attach-channel.md): Route logic externally
- [Configure module settings](partner-docs/module-technical-console.md): Console configuration
- [Chat control](partner-docs/module-technical-chat-control.md): Control which bot responds
- [Messaging API from module](partner-docs/module-technical-using-messaging-api.md): Send requests on behalf of bot

### LINE Notification Messages

- [Overview](partner-docs/line-notification-messages/overview.md): Send messages by phone number
- [Template](partner-docs/line-notification-messages/template.md): Template format and structure
- [Technical specs](partner-docs/line-notification-messages/technical-specs.md): Technical documentation
- [Webhook event](partner-docs/line-notification-messages/message-sending-complete-webhook-event.md): Delivery confirmation
- [Receiving flow](partner-docs/line-notification-messages/flow-when-receiving-message.md): User experience and flow

## API Reference

- [Messaging API](reference/messaging-api.md): Messaging API reference
- [Webhook Events](reference/webhook-events.md): Complete webhook event structures including Join, Member Joined/Left, Source objects
- [LINE Login](reference/line-login.md): LINE Login API reference
- [LIFF](reference/liff.md): LIFF SDK reference
- [LINE MINI App](reference/line-mini-app.md): LINE MINI App API reference
