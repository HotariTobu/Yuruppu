# Skipping the channel consent process

<!-- tip start -->

**This feature can only be used for verified MINI Apps**

This feature is only available for verified MINI Apps. For unverified MINI Apps, you can test the feature on the internal channel for Developing, but you can't use the feature on the internal channel for Published.

<!-- tip end -->

<!-- note start -->

**For new LINE MINI App channels in Japan, the use of the &quot;Channel consent simplification&quot; feature will be mandatory on January 8, 2026**

For more information, see the news from [October 31, 2025](https://developers.line.biz/en/news/2025/10/31/channel-consent-simplification/).

<!-- note end -->

<!-- note start -->

**Permission that is consented with the &quot;Channel consent simplification&quot; feature**

When users give [Consent on getting user profile information](https://developers.line.biz/en/docs/messaging-api/user-consent/), which is displayed when they first add a [LINE Official Account](https://developers.line.biz/en/glossary/#line-official-account) as a friend, it will be deemed that they've given consent to other LINE Official Accounts on getting their profile information, so the consent screen will be skipped thereafter. In the same way, when you enable the "Channel consent simplification" feature described in this page, once users agree to the simplification, they will be able to skip the "Channel consent screen" on subsequent LINE MINI Apps they access for the first time.

However, based on LY Corporation's privacy policy, the permission to skip the consent screen using the "Channel consent simplification" feature is limited to [user ID](https://developers.line.biz/en/glossary/#user-id) (the `openid` scope). Permission required to get user profile information or permission to send messages (such as [the `profile` scope and the `chat_message.write` scope](https://developers.line.biz/en/docs/liff/registering-liff-apps/#registering-liff-app)) aren't covered by "Channel consent simplification". Users will be prompted to give consent for these permissions when they become necessary within each LINE MINI App.

<!-- note end -->

When a user first accesses a LINE MINI App with the `openid` scope enabled, the [channel consent screen](https://developers.line.biz/en/docs/line-mini-app/develop/configure-console/#consent-screen-settings) is displayed where they need to consent to their [user ID](https://developers.line.biz/en/glossary/#user-id) being used within the LINE MINI App.

To simplify this consent process, turn on the "Channel consent simplification" feature for your LINE MINI App on the [LINE Developers Console](https://developers.line.biz/console/). This will allow users to skip the channel consent screen when accessing another LINE MINI App and start using the service immediately, simply by consenting to the simplification the first time.

Turning on the "Channel consent simplification" setting makes it easier for users to access LINE MINI Apps. We recommend enabling "Channel consent simplification" to reduce the burden on users.

<!-- tip start -->

**Operating conditions of &quot;Channel consent simplification&quot;**

"Channel consent simplification" operates only in these environments:

- Version of LIFF SDK for LINE MINI App: v2.13.x or later

<!-- tip end -->

<!-- note start -->

**LINE MINI App may not function properly depending on the design**

On the LINE MINI App channel, the only permission that is automatically granted by the "Channel consent simplification" feature is getting [user ID](https://developers.line.biz/en/glossary/#user-id). Therefore, if you've designed your channel to use the [access token](https://developers.line.biz/en/glossary/#access-token) obtained from the LIFF SDK to call the LINE Login API and other LINE APIs, the "Channel consent simplification" feature may not work properly.

Before starting to use the "Channel consent simplification" feature, check how the usage of access tokens applies in the case of the LINE MINI App, and confirm operation in the development environment.

**Example of impact:**<br>A design that uses the LIFF SDK to obtain an [ID token](https://developers.line.biz/en/glossary/#id-token) along with an access token, and uses the [profile information](https://developers.line.biz/en/glossary/#profile-information) (display name, email address, profile image, etc.) obtained through the ID token to create a LINE MINI App service account.

<!-- note end -->

#### Differences in behavior when the "Channel consent simplification" setting is On and Off 

Even if a user has already given their consent on the "Channel consent screen" when they first accessed a LINE MINI App, whenever they access another LINE MINI App for the first time, the same "Channel consent screen" will be displayed.

However, when a user gives their consent on the "Simplification consent screen" that is displayed when they access a LINE MINI App with the "Channel consent simplification" setting on, any LINE MINI App they access for the first time thereafter won't display the "Channel consent screen" but open soon after displaying a "loading screen".

The table below explains the differences in behavior when accessing a LINE MINI App with the "Channel consent simplification" setting **On** and **Off**.

| "Channel consent<br>simplification"<br>setting | When accessing LINE MINI App A for the first time | When accessing LINE MINI App B for the first time |
| :-: | :-: | :-: |
| **Off** | ![feature off (App A)](https://developers.line.biz/media/line-mini-app/channel-consent-simplification-disabled-app-a-en.png)<br> "Channel consent screen" is displayed. | ![feature off (App B)](https://developers.line.biz/media/line-mini-app/channel-consent-simplification-disabled-app-b-en.png)<br> "Channel consent screen" is displayed. |
| **On** | ![feature on (App A)](https://developers.line.biz/media/line-mini-app/channel-consent-simplification-enabled-app-a-en.png)<br> "Simplification consent screen" is displayed. | ![feature on (App B)](https://developers.line.biz/media/line-mini-app/channel-consent-simplification-enabled-app-b-en.png)<br> "Channel consent screen" is skipped. |

For the detailed workflow of accessing a LINE MINI App that has "Channel consent simplification" enabled, see [Detailed workflow of "Channel consent simplification"](https://developers.line.biz/en/docs/line-mini-app/develop/channel-consent-simplification/#detailed-workflow).

## The "Channel consent simplification" feature setup 

Follow these steps to turn on "Channel consent simplification".

1.  From the LINE MINI App channel on the [LINE Developers Console](https://developers.line.biz/console/), locate the **Channel consent simplification** section under the **Web app settings** tab and toggle the slider on (right).

    <!-- tip start -->

    **&quot;Channel consent simplification&quot; setting will be on by default**

    If you have created a new LINE MINI App channel, the **Channel consent simplification** setting will be on (right) by default. If you don't want to use "Channel consent simplification", you will have to turn it off (left).

    <!-- tip end -->

    ![Channel consent simplification toggle button](https://developers.line.biz/media/line-mini-app/simplification-feature-setup-en.png)

    <!-- note start -->

    **Conditions for configuring the &quot;Channel consent simplification&quot; feature**

    You can only configure the "Channel consent simplification" feature when these conditions are met.
    - LINE MINI App **Region to provide the service** is set to "Japan":

      Only those LINE MINI Apps with **Region to provide the service** set to "Japan" can configure this feature. **Region to provide the service** can only be configured when you first create your LINE MINI App channel.

      ![Region to provide the service settings](https://developers.line.biz/media/line-mini-app/region-setting-en.png)

    - LINE MINI App channel status is "Not yet reviewed":

    Only those LINE MINI Apps whose status is "Not yet reviewed" can configure this feature.

    ![Developing process](https://developers.line.biz/media/line-mini-app/simplification-developing-en.png)

    <!-- note end -->

2.  When the confirmation dialog is displayed, click **Enable**.

    ![confirm dialog](https://developers.line.biz/media/line-mini-app/simplification-dialog-en.png)

    <!-- note start -->

    **openid is automatically enabled**

    When using "Channel consent simplification", you need the `openid` scope, which has the authority to get user ID. When the "Channel consent simplification" setting is on, the `openid` scope will automatically be enabled. When the "Channel consent simplification" setting is turned off, you have the option of manually selecting the `openid` scope.

    ![openid scope setting](https://developers.line.biz/media/line-mini-app/simplification-scope-en.png)

    <!-- note end -->

## Detailed workflow of "Channel consent simplification" 

The first time a user accesses a LINE MINI App with the "Channel consent simplification" setting enabled, the "Simplification consent screen" will be displayed.

1. From the "Simplification consent screen", click **Allow**.

   ![Simplification consent screen](https://developers.line.biz/media/line-mini-app/simplification-process-01-en.png)

   When a user clicks **Allow**, it will be deemed that they agreed to the use of their [user ID](https://developers.line.biz/en/glossary/#user-id) in other LINE MINI Apps, so that when they access other LINE MINI Apps going forward, the "Channel consent screen" will be skipped, and the LINE MINI App will open immediately.

   <!-- tip start -->

   **When the &quot;Simplification consent screen&quot; appears again if you click &quot;Not now&quot;**

   By clicking **Not now** on the Simplification consent screen, the user can skip the consent for simplification, and the "Simplification consent screen" won't be displayed, even when they open other LINE MINI Apps. The Simplification consent screen will reappear once 24 hours have passed.

   If the user skips the consent for simplification, they will see a separate channel consent screen for each LINE MINI App they open, as they would if the "Channel consent simplification" feature were turned off.

   <!-- tip end -->

2. From the "loading screen", click **Open app now**.

   <!-- tip start -->

   **On the &quot;loading screen&quot;**

   - Even if the user doesn't click **Open app now** from the "loading screen", the LINE MINI App will be displayed without any user action, once the progress bar is complete.
   - After the user consents to the "Simplification consent screen", the "loading screen" will be displayed only once when the user accesses each LINE MINI App for the first time.

   <!-- tip end -->

   ![LINE MINI App loading screen](https://developers.line.biz/media/line-mini-app/simplification-process-02-en.png)

   LINE MINI App will be displayed.

3. Click **Allow** once the "Verification screen" is displayed.

   <!-- tip start -->

   **When the &quot;Verification screen&quot; is displayed**

   The "Verification screen" is first displayed, not when a user first opens a LINE MINI App, but when permission for scopes other than the `openid` scope ([the `profile` scope or the `chat_message.write`scope](https://developers.line.biz/en/docs/liff/registering-liff-apps/#registering-liff-app) etc.) is required.

   Therefore, if you've designed your LINE MINI App so that immediately after it's launched, it executes requests that require permissions other than the `openid` scope, such as the [`liff.getProfile()`](https://developers.line.biz/en/reference/liff/#get-profile) method, when users access your LINE MINI App, it would appear as if the channel consent screen were displayed without being skipped.

   <!-- tip end -->

   <!-- tip start -->

   **Display the &quot;Verification screen&quot; at any given time**

   By using the [`liff.permission.query()`](https://developers.line.biz/en/reference/liff/#permission-query) method and the [`liff.permission.requestAll()`](https://developers.line.biz/en/reference/liff/#permission-request-all) method, you can display the "Verification screen" at any given time.

   The following is an example code that displays the "verification screen" when the user hasn't consented to grant permissions in the `profile` scope.

   ```javascript
   liff.permission.query("profile").then((permissionStatus) => {
     if (permissionStatus.state === "prompt") {
       liff.permission.requestAll();
     }
   });
   ```

   For more information, see [`liff.permission.query()`](https://developers.line.biz/en/reference/liff/#permission-query) and [`liff.permission.requestAll()`](https://developers.line.biz/en/reference/liff/#permission-request-all) in the LIFF API reference.

   <!-- tip end -->

   Check the verification for each scope, and click **Allow** to open the LINE MINI App.

   ![verification screen](https://developers.line.biz/media/line-mini-app/simplification-process-03-en.png)

Users who have followed the above steps will be able to skip the channel consent screen, even for LINE MINI Apps that they're accessing for the first time, and open LINE MINI Apps immediately after the "loading screen" is displayed.

![Channel consent simplification enabled](https://developers.line.biz/media/line-mini-app/channel-consent-simplification-enabled-en.png)

### Channel consent simplification doesn't work for LINE MINI Apps opened in LIFF-to-LIFF transitions 

Channel consent simplification doesn't work when users transition to a LINE MINI App from a LIFF app or another LINE MINI App. Even if Channel consent simplification is enabled on the LINE MINI Apps to which users are transitioning, an individual "Channel consent screen" will be displayed for every LINE MINI App on the first access.

For more information on LIFF-to-LIFF transition, see [Opening a LIFF app from another LIFF app (LIFF-to-LIFF transition)](https://developers.line.biz/en/docs/liff/opening-liff-app/#move-liff-to-liff).

## Important points about using the "Channel consent simplification" feature together with the add friend option 

In the LINE MINI App, you can use the [add friend option](https://developers.line.biz/en/docs/line-mini-app/service/line-mini-app-oa/#link-a-line-official-account-with-your-channel) to prompt users to add your LINE Official Account from the channel consent screen or the verification screen.

![](https://developers.line.biz/media/line-mini-app/channel-consent-simplification/line-mini-app-playground-channel-consent-screen-en.png) ![](https://developers.line.biz/media/line-mini-app/channel-consent-simplification/line-mini-app-playground-verification-screen-en.png)

However, if only `openid` is specified in the "Scope" section of the **Web app settings** tab in your LINE MINI App channel, enabling the "Channel consent simplification" feature will prevent you from prompting users to add friends using the add friend option.

When using the "Channel consent simplification" feature together with the add friend option, we recommend specifying scopes other than `openid` in the "Scope" section of the **Web app settings** tab in your LINE MINI App channel, and displaying the verification screen using one of the following methods:

- [Method 1. Use the `liff.permission.query()` method and the `liff.permission.requestAll()` method](https://developers.line.biz/en/docs/line-mini-app/develop/channel-consent-simplification/#add-friend-option-method1)
- [Method 2. Use methods that require permissions other than the `openid` scope](https://developers.line.biz/en/docs/line-mini-app/develop/channel-consent-simplification/#add-friend-option-method2)

### Method 1. Use the `liff.permission.query()` method and the `liff.permission.requestAll()` method 

You can use the [`liff.permission.query()`](https://developers.line.biz/en/reference/liff/#permission-query) method and the [`liff.permission.requestAll()`](https://developers.line.biz/en/reference/liff/#permission-request-all) method to display the verification screen.

```javascript
liff.permission.query("profile").then((permissionStatus) => {
  if (permissionStatus.state === "prompt") {
    liff.permission.requestAll();
  }
});
```

For more information, see [`liff.permission.query()`](https://developers.line.biz/en/reference/liff/#permission-query) and [`liff.permission.requestAll()`](https://developers.line.biz/en/reference/liff/#permission-request-all) in the LIFF API reference.

### Method 2. Use methods that require permissions other than the `openid` scope 

You can use methods that require permissions other than the `openid` scope to display the verification screen. The following methods require permissions other than the `openid` scope:

| Scope | Method |
| --- | --- |
| `email` | <ul><li>[`liff.getIDToken()`](https://developers.line.biz/en/reference/liff/#get-id-token)</li><li>[`liff.getDecodedIDToken()`](https://developers.line.biz/en/reference/liff/#get-decoded-id-token)</li></ul> |
| `profile` | <ul><li>[`liff.getProfile()`](https://developers.line.biz/en/reference/liff/#get-profile)</li><li>[`liff.getFriendship()`](https://developers.line.biz/en/reference/liff/#get-friendship)</li></ul> |
| `chat_message.write` | <ul><li>[`liff.sendMessages()`](https://developers.line.biz/en/reference/liff/#send-messages)</li></ul> |
