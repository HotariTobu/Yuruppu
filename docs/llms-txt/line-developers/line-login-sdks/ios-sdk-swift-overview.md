# LINE SDK for iOS Swift - Overview

## Overview

The LINE SDK for iOS Swift is a modern, open-source framework enabling iOS developers to integrate LINE Login and related features into their applications.

## Key Features

### User Authentication
"Users to log in to your app or service with their LINE accounts" without entering credentials if already logged into LINE on their device.

### OpenID Connect Support
The SDK implements "OpenID Connect 1.0 specification" to provide ID tokens containing user profile data, eliminating the need for separate user registration systems.

### API Methods
Developers can retrieve user profiles, manage sessions, and handle access token operations through built-in SDK methods.

## Getting Started Steps

1. **Create a channel** via LINE Developers console for LINE Login access
2. **Integrate the SDK** into your project (see Setting up your project documentation)
3. **Implement LINE Login** in your application
4. **Manage user sessions** and access tokens

## Documentation Structure

The complete guide covers:

| Topic | Purpose |
|-------|---------|
| Setting up your project | SDK integration instructions |
| Integrating LINE Login | Implementation guidance |
| Managing users | Profile retrieval and logout |
| Managing access tokens | Token refresh and verification |
| Handling errors | Error management patterns |
| Objective-C compatibility | Legacy code support |
| Migration guide | Upgrading from v4.1 to v5 |

## Resources

- **GitHub Repository**: Official open-source code and samples available
- **Starter App**: Reference implementation for testing LOGIN functionality
- **API Reference**: Detailed protocol and class documentation at `/en/reference/ios-sdk-swift/`
