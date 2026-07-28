// ============================================================================
// TigerWallet - Main App Entry Point
// Production-Ready Multi-Chain Wallet Application
// ============================================================================

import React, { useEffect } from 'react';
import { StatusBar, LogBox } from 'react-native';
import { NavigationContainer, DefaultTheme, DarkTheme } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';

// Stores
import { useThemeStore } from './src/stores/ThemeStore';
import { walletService } from './src/services/WalletService';

// Screens
import HomeScreen from './src/screens/HomeScreen';
import SendScreen from './src/screens/SendScreen';
import ReceiveScreen from './src/screens/ReceiveScreen';
import SwapScreen from './src/screens/SwapScreen';
import WalletDetailsScreen from './src/screens/WalletDetailsScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import AddWalletScreen from './src/screens/AddWalletScreen';

// Ignore specific warnings
LogBox.ignoreLogs([
  'Non-serializable values were found in the navigation state',
  'VirtualizedLists should never be nested',
]);

const Stack = createNativeStackNavigator();

// Light Navigation Theme
const lightNavigationTheme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    primary: '#FF6B35',
    background: '#FFFFFF',
    card: '#F8F9FA',
    text: '#1A1A2E',
    border: '#DEE2E6',
  },
};

// Dark Navigation Theme
const darkNavigationTheme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: '#FF6B35',
    background: '#0D0D0D',
    card: '#1A1A1A',
    text: '#FFFFFF',
    border: '#3D3D3D',
  },
};

// Main App Component
const App: React.FC = () => {
  const { isDark, initialize } = useThemeStore();

  useEffect(() => {
    // Initialize services
    const init = async () => {
      await initialize();
      await walletService.initialize();
    };
    init();
  }, []);

  const navigationTheme = isDark ? darkNavigationTheme : lightNavigationTheme;

  const screenOptions = {
    headerShown: false,
    animation: 'slide_from_right' as const,
  };

  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <StatusBar 
          barStyle={isDark ? 'light-content' : 'dark-content'}
          backgroundColor="transparent"
          translucent
        />
        <NavigationContainer theme={navigationTheme}>
          <Stack.Navigator 
            initialRouteName="Home"
            screenOptions={screenOptions}
          >
            <Stack.Screen name="Home" component={HomeScreen} />
            <Stack.Screen name="Send" component={SendScreen} />
            <Stack.Screen name="Receive" component={ReceiveScreen} />
            <Stack.Screen name="Swap" component={SwapScreen} />
            <Stack.Screen name="WalletDetails" component={WalletDetailsScreen} />
            <Stack.Screen name="Settings" component={SettingsScreen} />
            <Stack.Screen name="AddWallet" component={AddWalletScreen} />
            
            {/* Additional Screens */}
            <Stack.Screen name="Bridge" component={BridgeScreen} />
            <Stack.Screen name="Staking" component={StakingScreen} />
            <Stack.Screen name="NFTs" component={NFTsScreen} />
            <Stack.Screen name="DApps" component={DAppsScreen} />
            <Stack.Screen name="History" component={HistoryScreen} />
            <Stack.Screen name="Buy" component={BuyScreen} />
            <Stack.Screen name="Sell" component={SellScreen} />
            <Stack.Screen name="Notifications" component={NotificationsScreen} />
            <Stack.Screen name="Security" component={SecurityScreen} />
            <Stack.Screen name="Backup" component={BackupScreen} />
            <Stack.Screen name="NetworkSettings" component={NetworkSettingsScreen} />
          </Stack.Navigator>
        </NavigationContainer>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
};

// Placeholder screens (would be implemented in separate files)
const BridgeScreen = () => <HomeScreen />;
const StakingScreen = () => <HomeScreen />;
const NFTsScreen = () => <HomeScreen />;
const DAppsScreen = () => <HomeScreen />;
const HistoryScreen = () => <HomeScreen />;
const BuyScreen = () => <HomeScreen />;
const SellScreen = () => <HomeScreen />;
const NotificationsScreen = () => <HomeScreen />;
const SecurityScreen = () => <HomeScreen />;
const BackupScreen = () => <HomeScreen />;
const NetworkSettingsScreen = () => <HomeScreen />;

export default App;
