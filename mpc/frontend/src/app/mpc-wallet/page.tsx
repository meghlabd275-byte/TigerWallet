"use client";

import { useState, useCallback, useEffect } from "react";
import { 
  Shield, 
  Key, 
  Smartphone, 
  Copy, 
  CheckCircle,
  AlertCircle,
  Loader2,
  Plus,
  Trash2,
  Fingerprint,
  Eye,
  EyeOff,
  ArrowRight,
  Lock,
  Unlock,
  RefreshCw
} from "lucide-react";
import { BrowserProvider, JsonRpcSigner } from "ethers";

// MPC Configuration
const MPC_CONFIG = {
  threshold: 2,
  totalShares: 3,
  serverUrl: "https://api.mpc.tigerwallet.io",
  sessionDuration: 3600, // 1 hour
};

// Types
interface MPCKeyShare {
  id: string;
  type: "device" | "server" | "backup";
  encryptedData: string;
  createdAt: number;
  name?: string;
}

interface MPCWallet {
  address: string;
  publicKey: string;
  shares: MPCKeyShare[];
  isSetup: boolean;
  backupCreated: boolean;
}

interface RecoveryContact {
  id: string;
  name: string;
  publicKey: string;
  addedAt: number;
}

// Social Login Providers
const SOCIAL_PROVIDERS = [
  { id: "google", name: "Google", icon: "G", color: "#4285F4" },
  { id: "apple", name: "Apple", icon: "🍎", color: "#000000" },
  { id: "email", name: "Email", icon: "📧", color: "#EA4335" },
];

export default function MPCWalletPage() {
  const [wallet, setWallet] = useState<MPCWallet | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [step, setStep] = useState<"intro" | "setup" | "social" | "backup" | "complete">("intro");
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [recoveryContacts, setRecoveryContacts] = useState<RecoveryContact[]>([]);
  const [newContactName, setNewContactName] = useState("");
  const [showSeed, setShowSeed] = useState(false);
  const [sessionActive, setSessionActive] = useState(false);
  const [sessionTimeLeft, setSessionTimeLeft] = useState(0);

  // Generate MPC key shares
  const generateKeyShares = useCallback(async (): Promise<MPCKeyShare[]> => {
    setIsLoading(true);
    try {
      // In production, this would use a secure MPC library
      // For demo, we simulate key generation
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const shares: MPCKeyShare[] = [
        {
          id: "device-1",
          type: "device",
          encryptedData: await encryptShare("device-share-" + Date.now()),
          createdAt: Date.now(),
          name: "This Device",
        },
        {
          id: "server-1",
          type: "server",
          encryptedData: await encryptShare("server-share-" + Date.now()),
          createdAt: Date.now(),
          name: "TigerWallet Server",
        },
        {
          id: "backup-1",
          type: "backup",
          encryptedData: await encryptShare("backup-share-" + Date.now()),
          createdAt: Date.now(),
          name: "Backup Key",
        },
      ];
      
      return shares;
    } catch (err: any) {
      throw new Error("Failed to generate key shares");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Simple encryption for demo
  const encryptShare = async (data: string): Promise<string> => {
    // In production, use proper encryption
    return btoa(data);
  };

  // Setup MPC Wallet
  const setupWallet = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const shares = await generateKeyShares();
      
      // Derive address from public key
      const publicKey = "0x" + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join("");
      const address = "0x" + Array(40).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join("");
      
      setWallet({
        address,
        publicKey,
        shares,
        isSetup: true,
        backupCreated: false,
      });
      
      setStep("social");
    } catch (err: any) {
      setError(err.message || "Failed to setup wallet");
    } finally {
      setIsLoading(false);
    }
  }, [generateKeyShares]);

  // Social Login
  const handleSocialLogin = useCallback(async (provider: string) => {
    setIsLoading(true);
    setError(null);
    setSelectedProvider(provider);
    
    try {
      // Simulate OAuth flow
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // In production, would get OAuth token and create MPC share
      setSuccess(`Successfully authenticated with ${provider}!`);
      setStep("backup");
    } catch (err: any) {
      setError("Authentication failed. Please try again.");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create Backup
  const createBackup = useCallback(async () => {
    if (!wallet) return;
    
    setIsLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setWallet({
        ...wallet,
        backupCreated: true,
      });
      
      setStep("complete");
    } catch (err: any) {
      setError("Failed to create backup");
    } finally {
      setIsLoading(false);
    }
  }, [wallet]);

  // Add Recovery Contact
  const addRecoveryContact = useCallback(async () => {
    if (!newContactName.trim()) return;
    
    const contact: RecoveryContact = {
      id: "contact-" + Date.now(),
      name: newContactName,
      publicKey: "0x" + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(""),
      addedAt: Date.now(),
    };
    
    setRecoveryContacts(prev => [...prev, contact]);
    setNewContactName("");
  }, [newContactName]);

  // Remove Recovery Contact
  const removeContact = useCallback((id: string) => {
    setRecoveryContacts(prev => prev.filter(c => c.id !== id));
  }, []);

  // Start Session
  const startSession = useCallback(async () => {
    if (!wallet) return;
    
    setIsLoading(true);
    try {
      // Simulate session creation
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setSessionActive(true);
      setSessionTimeLeft(MPC_CONFIG.sessionDuration);
      setSuccess("Session started successfully!");
    } catch (err: any) {
      setError("Failed to start session");
    } finally {
      setIsLoading(false);
    }
  }, [wallet]);

  // End Session
  const endSession = useCallback(() => {
    setSessionActive(false);
    setSessionTimeLeft(0);
    setSuccess("Session ended");
  }, []);

  // Session countdown
  useEffect(() => {
    if (!sessionActive || sessionTimeLeft <= 0) return;
    
    const timer = setInterval(() => {
      setSessionTimeLeft(prev => {
        if (prev <= 1) {
          setSessionActive(false);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    
    return () => clearInterval(timer);
  }, [sessionActive, sessionTimeLeft]);

  // Render intro screen
  if (step === "intro") {
    return (
      <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
        <div className="max-w-4xl mx-auto">
          <header className="flex items-center gap-3 mb-12">
            <div className="w-12 h-12 bg-gradient-to-br from-orange-500 to-orange-600 rounded-xl flex items-center justify-center">
              <Shield className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">MPC Security</p>
            </div>
          </header>

          <div className="text-center py-16">
            <div className="w-32 h-32 mx-auto mb-8 bg-gray-800 rounded-full flex items-center justify-center">
              <Key className="w-16 h-16 text-orange-500" />
            </div>
            
            <h2 className="text-4xl font-bold mb-4">MPC Keyless Wallet</h2>
            <p className="text-gray-400 text-lg mb-8 max-w-2xl mx-auto">
              Your private key is split into multiple shares. No seed phrase to lose, 
              no single point of failure. Access your wallet with your device, biometrics, or trusted contacts.
            </p>

            {/* Features */}
            <div className="grid md:grid-cols-3 gap-6 mb-12 text-left">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Fingerprint className="w-10 h-10 text-cyan-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">Biometric Access</h3>
                <p className="text-gray-400">Use Face ID or Touch ID to access your wallet</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Smartphone className="w-10 h-10 text-cyan-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">Social Recovery</h3>
                <p className="text-gray-400">Recover with trusted contacts or social accounts</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Lock className="w-10 h-10 text-cyan-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">No Seed Phrase</h3>
                <p className="text-gray-400">Your keys are never exposed in full</p>
              </div>
            </div>

            <button
              onClick={setupWallet}
              disabled={isLoading}
              className="bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 px-8 py-4 rounded-xl font-bold text-lg flex items-center gap-3 mx-auto transition-all hover:scale-105"
            >
              {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Shield className="w-5 h-5" />}
              Create MPC Wallet
            </button>
            
            {error && (
              <div className="mt-4 bg-red-500/10 border border-red-500/50 rounded-lg p-4 max-w-md mx-auto">
                <p className="text-red-400">{error}</p>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // Render social login screen
  if (step === "social") {
    return (
      <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
        <div className="max-w-2xl mx-auto">
          <header className="flex items-center gap-3 mb-8">
            <button onClick={() => setStep("intro")} className="text-gray-400 hover:text-white">
              ← Back
            </button>
          </header>

          <div className="text-center mb-8">
            <h2 className="text-3xl font-bold mb-2">Add Authentication</h2>
            <p className="text-gray-400">Link a social account for easy recovery</p>
          </div>

          <div className="space-y-4">
            {SOCIAL_PROVIDERS.map(provider => (
              <button
                key={provider.id}
                onClick={() => handleSocialLogin(provider.id)}
                disabled={isLoading}
                className="w-full flex items-center justify-between p-4 bg-gray-800 border border-gray-700 rounded-xl hover:border-gray-600 transition-colors"
              >
                <div className="flex items-center gap-4">
                  <div 
                    className="w-12 h-12 rounded-full flex items-center justify-center text-xl font-bold"
                    style={{ backgroundColor: provider.color }}
                  >
                    {provider.icon}
                  </div>
                  <span className="text-lg font-medium">Continue with {provider.name}</span>
                </div>
                <ArrowRight className="w-5 h-5 text-gray-400" />
              </button>
            ))}
          </div>

          {error && (
            <div className="mt-6 bg-red-500/10 border border-red-500/50 rounded-lg p-4">
              <p className="text-red-400">{error}</p>
            </div>
          )}
          
          {success && (
            <div className="mt-6 bg-green-500/10 border border-green-500/50 rounded-lg p-4">
              <p className="text-green-400">{success}</p>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Render backup screen
  if (step === "backup") {
    return (
      <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
        <div className="max-w-2xl mx-auto">
          <header className="flex items-center gap-3 mb-8">
            <button onClick={() => setStep("social")} className="text-gray-400 hover:text-white">
              ← Back
            </button>
          </header>

          <div className="text-center mb-8">
            <h2 className="text-3xl font-bold mb-2">Create Backup</h2>
            <p className="text-gray-400">Set up recovery contacts for backup access</p>
          </div>

          {/* Recovery Contacts */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 mb-6">
            <h3 className="text-xl font-bold mb-4">Recovery Contacts</h3>
            
            {recoveryContacts.length > 0 ? (
              <div className="space-y-3 mb-4">
                {recoveryContacts.map(contact => (
                  <div key={contact.id} className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                    <div>
                      <p className="font-medium">{contact.name}</p>
                      <p className="text-gray-400 text-sm font-mono">{contact.publicKey.slice(0, 10)}...</p>
                    </div>
                    <button
                      onClick={() => removeContact(contact.id)}
                      className="p-2 hover:bg-red-500/20 rounded-lg text-red-400"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-400 mb-4">No recovery contacts added yet</p>
            )}
            
            <div className="flex gap-2">
              <input
                type="text"
                value={newContactName}
                onChange={(e) => setNewContactName(e.target.value)}
                placeholder="Contact name..."
                className="flex-1 bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-orange-500"
              />
              <button
                onClick={addRecoveryContact}
                className="bg-orange-500 hover:bg-orange-600 px-4 py-2 rounded-lg flex items-center gap-2"
              >
                <Plus className="w-4 h-4" /> Add
              </button>
            </div>
          </div>

          {/* Backup Info */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 mb-6">
            <h3 className="text-xl font-bold mb-4">Backup Keys</h3>
            <p className="text-gray-400 mb-4">
              Your wallet has {MPC_CONFIG.totalShares} key shares. You need {MPC_CONFIG.threshold} to sign transactions.
            </p>
            
            <div className="space-y-3">
              {wallet?.shares.map((share, i) => (
                <div key={share.id} className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-orange-500/20 rounded-full flex items-center justify-center">
                      <Key className="w-4 h-4 text-orange-500" />
                    </div>
                    <span>{share.name}</span>
                  </div>
                  <span className="text-gray-400 text-sm capitalize">{share.type}</span>
                </div>
              ))}
            </div>
          </div>

          <button
            onClick={createBackup}
            disabled={isLoading}
            className="w-full bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 py-4 rounded-xl font-bold text-lg flex items-center justify-center gap-3"
          >
            {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : <CheckCircle className="w-5 h-5" />}
            Complete Setup
          </button>
        </div>
      </div>
    );
  }

  // Render complete/main screen
  return (
    <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <header className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-orange-500 to-orange-600 rounded-xl flex items-center justify-center">
              <Shield className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">MPC Security</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            {sessionActive ? (
              <div className="flex items-center gap-3">
                <span className="text-sm text-gray-400">
                  Session: {Math.floor(sessionTimeLeft / 60)}:{(sessionTimeLeft % 60).toString().padStart(2, '0')}
                </span>
                <button
                  onClick={endSession}
                  className="bg-red-500/20 hover:bg-red-500/30 border border-red-500/50 px-4 py-2 rounded-lg text-red-400 flex items-center gap-2"
                >
                  <Lock className="w-4 h-4" /> Lock
                </button>
              </div>
            ) : (
              <button
                onClick={startSession}
                disabled={isLoading}
                className="bg-green-500/20 hover:bg-green-500/30 border border-green-500/50 px-4 py-2 rounded-lg text-green-400 flex items-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Unlock className="w-4 h-4" />}
                Unlock
              </button>
            )}
          </div>
        </header>

        {error && (
          <div className="mb-6 bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-500" />
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {success && (
          <div className="mb-6 bg-green-500/10 border border-green-500/50 rounded-lg p-4 flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-500" />
            <p className="text-green-400">{success}</p>
          </div>
        )}

        {/* Wallet Status */}
        <div className="grid md:grid-cols-2 gap-6">
          {/* Wallet Info */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
            <h2 className="text-xl font-bold mb-4">Wallet Address</h2>
            
            <div className="bg-gray-900/50 rounded-lg p-4 mb-4">
              <p className="font-mono text-sm break-all">{wallet?.address}</p>
            </div>
            
            <button
              onClick={() => navigator.clipboard.writeText(wallet?.address || "")}
              className="flex items-center gap-2 text-gray-400 hover:text-white"
            >
              <Copy className="w-4 h-4" /> Copy Address
            </button>
          </div>

          {/* Security Status */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
            <h2 className="text-xl font-bold mb-4">Security Status</h2>
            
            <div className="space-y-3">
              <div className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                <span className="text-gray-400">Key Shares</span>
                <span className="text-green-400">{wallet?.shares.length}/{MPC_CONFIG.totalShares}</span>
              </div>
              <div className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                <span className="text-gray-400">Recovery Contacts</span>
                <span className="text-cyan-400">{recoveryContacts.length}</span>
              </div>
              <div className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                <span className="text-gray-400">Social Login</span>
                <span className="text-green-400 flex items-center gap-1">
                  <CheckCircle className="w-4 h-4" /> Active
                </span>
              </div>
            </div>
          </div>

          {/* Key Shares */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 md:col-span-2">
            <h2 className="text-xl font-bold mb-4">Your Key Shares</h2>
            
            <div className="grid md:grid-cols-3 gap-4">
              {wallet?.shares.map(share => (
                <div key={share.id} className="bg-gray-900/50 rounded-lg p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <Key className="w-5 h-5 text-orange-500" />
                    <span className="font-medium">{share.name}</span>
                  </div>
                  <p className="text-gray-400 text-sm capitalize">{share.type}</p>
                  <p className="text-gray-500 text-xs mt-2">
                    Created: {new Date(share.createdAt).toLocaleDateString()}
                  </p>
                </div>
              ))}
            </div>
          </div>

          {/* Recovery Contacts */}
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 md:col-span-2">
            <h2 className="text-xl font-bold mb-4">Recovery Contacts</h2>
            
            {recoveryContacts.length > 0 ? (
              <div className="grid md:grid-cols-2 gap-4">
                {recoveryContacts.map(contact => (
                  <div key={contact.id} className="bg-gray-900/50 rounded-lg p-4 flex items-center justify-between">
                    <div>
                      <p className="font-medium">{contact.name}</p>
                      <p className="text-gray-400 text-sm font-mono">{contact.publicKey.slice(0, 16)}...</p>
                    </div>
                    <button
                      onClick={() => removeContact(contact.id)}
                      className="p-2 hover:bg-red-500/20 rounded-lg text-red-400"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-400 text-center py-8">No recovery contacts added</p>
            )}
            
            <div className="mt-4 flex gap-2">
              <input
                type="text"
                value={newContactName}
                onChange={(e) => setNewContactName(e.target.value)}
                placeholder="Add recovery contact name..."
                className="flex-1 bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-orange-500"
              />
              <button
                onClick={addRecoveryContact}
                className="bg-orange-500 hover:bg-orange-600 px-4 py-2 rounded-lg flex items-center gap-2"
              >
                <Plus className="w-4 h-4" /> Add
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
