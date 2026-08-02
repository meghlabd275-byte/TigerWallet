# TigerWallet Passkeys/WebAuthn Authentication

## Overview

This module provides passwordless authentication using WebAuthn (Passkeys) for all TigerWallet platforms. Passkeys are more secure than passwords and eliminate phishing risks.

## Features

- **Passwordless Login**: Use device biometrics or PIN
- **Cross-Device Sync**: Passkeys sync across devices
- **Phishing Resistant**: Bound to specific domains
- **Hardware Security**: Use TPM/HSM when available
- **Multi-Device**: Support for phones, tablets, desktops

## Platform Support

| Platform | Support |
|----------|---------|
| Android | ✅ Biometric + PIN |
| iOS | ✅ Face ID + Touch ID |
| Web | ✅ WebAuthn API |
| Desktop | ✅ Biometric + PIN |

## Installation

```bash
npm install @tigerwallet/passkeys
```

## Web Usage

```typescript
import { PasskeysAuth } from '@tigerwallet/passkeys';

const passkeys = new PasskeysAuth({
  relyingPartyId: 'tigerwallet.com',
  relyingPartyName: 'TigerWallet',
});

// Register new passkey
async function register() {
  const { credential, publicKey } = await passkeys.createCredential({
    user: {
      id: 'user123',
      name: 'user@tigerwallet.com',
      displayName: 'Tiger User',
    },
    authenticator: {
      residentKey: 'required',
      userVerification: 'preferred',
    },
  });
  
  // Store credential ID for login
  await saveCredentialToServer(credential);
}

// Login with passkey
async function login() {
  const assertion = await passkeys.getCredential({
    challenge: await getChallengeFromServer(),
    allowCredentials: [
      { type: 'public-key', id: credentialId },
    ],
  });
  
  // Verify on server
  await verifyAssertionOnServer(assertion);
}
```

## Android Implementation

```kotlin
class PasskeyService {
    
    // Generate registration options
    fun generateRegistrationOptions(userId: String): PublicKeyCredentialCreationOptions {
        return PublicKeyCredentialCreationOptions(
            rp = RelyingParty(
                id = "tigerwallet.com",
                name = "TigerWallet"
            ),
            user = UserEntity(
                id = userId.toByteArray(),
                name = "user@tigerwallet.com",
                displayName = "Tiger User"
            ),
            challenge = generateChallenge(),
            pubKeyCredParams = listOf(
                PublicKeyCredentialParam(Alg.ES256),
                PublicKeyCredentialParam(Alg.RS256)
            ),
            authenticatorSelection = AuthenticatorSelectionCriteria(
                residentKey = ResidentKeyRequirement.REQUIRED,
                userVerification = UserVerificationRequirement.PREFERRED
            )
        )
    }
    
    // Register passkey
    suspend fun registerPasskey(
        context: Context,
        options: PublicKeyCredentialCreationOptions,
        prompt: BiometricPrompt
    ): ByteArray {
        val credentialManager = CredentialManager.create(context)
        
        val createResult = credentialManager.createCredential(
            request = CreatePublicKeyCredentialRequest(
                json = options.toJson(),
                isSystemProvider = true
            ),
            cancellationSignal = null,
            executor = ContextCompat.getMainExecutor(context)
        ) as CreatePasskeyResult
        
        return createResult.authenticationResponse
    }
    
    // Authenticate with passkey
    suspend fun authenticate(
        challenge: ByteArray,
        credentialId: ByteArray
    ): ByteArray {
        // Use Android's Credential Manager
    }
}
```

## iOS Implementation

```swift
class PasskeyService {
    
    // Register passkey
    func registerPasskey(userId: String, completion: @escaping (Result<Data, Error>) -> Void) {
        let relyingPartyId = "tigerwallet.com"
        
        let registration = ASAuthorizationPublicKeyCredentialRegistration()
        registration.userId = userId.data(using: .utf8)!
        registration.challenge = generateChallenge()
        relyingPartyIdentifier = relyingPartyId
        
        let authController = ASAuthorizationController(authorizationRequests: [registration])
        authController.delegate = self
        authController.performRequests()
    }
    
    // Login with passkey
    func loginWithPasskey(credentialId: Data, completion: @escaping (Result<Data, Error>) -> Void) {
        let assertion = ASAuthorizationPublicKeyCredentialAssertion()
        assertion.credentialId = credentialId
        assertion.challenge = generateChallenge()
        relyingPartyIdentifier = "tigerwallet.com"
        
        let authController = ASAuthorizationController(authorizationRequests: [assertion])
        authController.delegate = self
        authController.performRequests()
    }
}
```

## Flutter Implementation

```dart
class PasskeyService {
  static const _plugin = PasskeysPlugin();
  
  // Create credential
  Future<PasskeyCredential> createPasskey({
    required String userId,
    required String username,
  }) async {
    return await _plugin.createCredential(
      relyingPartyId: 'tigerwallet.com',
      relyingPartyName: 'TigerWallet',
      user: UserPasskeyInfo(
        id: userId,
        name: username,
        displayName: username,
      ),
      authenticatorSelection: AuthenticatorSelection(
        residentKey: ResidentKeyRequirement.required,
        userVerification: UserVerificationPreference.preferred,
      ),
    );
  }
  
  // Get credential
  Future<PasskeyAssertion> getPasskey({
    required String challenge,
    required List<String> allowedCredentialIds,
  }) async {
    return await _plugin.getCredential(
      challenge: challenge,
      relyingPartyId: 'tigerwallet.com',
      allowedCredentials: allowedCredentialIds.map((id) => {
        'type': 'public-key',
        'id': id,
      }).toList(),
    );
  }
}
```

## React Native Implementation

```typescript
// PasskeyService.ts
import * as Keychain from 'react-native-keychain';

class PasskeyService {
  
  // Generate key pair for passkey
  async generateKeyPair(): Promise<{ publicKey: string; privateKey: string }> {
    // Use secure enclave or keychain
  }
  
  // Store passkey credentials
  async storeCredential(credentialId: string, publicKey: string): Promise<void> {
    await Keychain.setGenericPassword(credentialId, publicKey, {
      service: 'tigerwallet.passkeys',
      accessControl: Keychain.ACCESS_CONTROL.BIOMETRY_ANY_OR_DEVICE_PASSCODE,
    });
  }
  
  // Get passkey credentials
  async getCredentials(): Promise<Array<{ id: string; publicKey: string }>> {
    const credentials = await Keychain.getGenericPassword({
      service: 'tigerwallet.passkeys',
    });
    // Parse and return
  }
  
  // Sign with passkey
  async sign(challenge: string, privateKey: string): Promise<string> {
    // Sign challenge with private key
  }
}
```

## Server-Side Verification

```typescript
// verify-passkey.ts
import { verifyAuthentication } from '@tigerwallet/passkeys';

async function verifyLogin(request: Request) {
  const { credential, challenge } = await request.json();
  
  // Verify the assertion
  const result = await verifyAuthentication({
    credential: credential,
    expectedChallenge: challenge,
    expectedOrigin: 'https://tigerwallet.com',
    expectedRpId: 'tigerwallet.com',
    supportedAlgorithms: ['ES256', 'RS256'],
  });
  
  if (result.verified) {
    // Create session
    const session = await createSession(result.userId);
    return { sessionToken: session.token };
  } else {
    throw new Error('Authentication failed');
  }
}
```

## Security Features

### 1. Phishing Protection
- Passkeys are bound to specific domain
- Cannot be used on phishing sites
- Verified hostname during authentication

### 2. Device Binding
- Hardware-backed when available
- TPM/Secure Enclave integration
- Device-specific credentials

### 3. User Verification
- Biometric required when configured
- Fallback to device PIN
- Two-factor built-in

### 4. Credential Storage
- OS-level secure storage
- Encrypted at rest
- Hardware encryption when available

## API Reference

### PasskeysAuth Class

```typescript
class PasskeysAuth {
  constructor(config: PasskeysConfig);
  
  // Create new passkey credential
  async createCredential(options: CreateCredentialOptions): Promise<Credential>;
  
  // Get existing credential (login)
  async getCredential(options: GetCredentialOptions): Promise<Assertion>;
  
  // Remove credential
  async removeCredential(credentialId: string): Promise<void>;
  
  // List all credentials
  async listCredentials(): Promise<CredentialInfo[]>;
}
```

### Types

```typescript
interface PasskeysConfig {
  relyingPartyId: string;
  relyingPartyName: string;
  origin?: string;
}

interface CreateCredentialOptions {
  user: {
    id: string;
    name: string;
    displayName: string;
  };
  authenticator?: {
    residentKey?: 'required' | 'preferred' | 'discouraged';
    userVerification?: 'required' | 'preferred' | 'discouraged';
    authenticatorAttachment?: 'platform' | 'cross-platform';
  };
}

interface GetCredentialOptions {
  challenge: string;
  allowCredentials?: Array<{
    type: 'public-key';
    id: string;
  }>;
  userVerification?: 'required' | 'preferred' | 'discouraged';
}
```

## Migration from Passwords

```typescript
// Allow both password and passkey during transition
class AuthService {
  async login(credentials: LoginCredentials) {
    // Try passkey first
    if (credentials.passkeyAssertion) {
      const passkeyResult = await this.passkeys.verify(credentials.passkeyAssertion);
      if (passkeyResult.success) {
        return this.createSession(passkeyResult.userId);
      }
    }
    
    // Fall back to password
    if (credentials.password) {
      const passwordResult = await this.verifyPassword(credentials);
      if (passwordResult.success) {
        // Offer to create passkey
        await this.offerPasskeyEnrollment(passwordResult.userId);
        return this.createSession(passwordResult.userId);
      }
    }
    
    throw new Error('Invalid credentials');
  }
}
```

## Best Practices

1. **Always Offer Passkeys**: Enable passkey creation after login
2. **Support Multiple**: Allow multiple passkeys per account
3. **Cross-Device**: Enable sync across user's devices
4. **Backup**: Offer backup methods for account recovery
5. **Clear UI**: Explain benefits of passkeys to users
