/**
 * TigerWallet Admin - Logger Utility
 */

import 'package:flutter/foundation.dart';

class Logger {
  static bool _initialized = false;
  
  static void init() {
    _initialized = true;
  }
  
  static void d(String message) {
    if (_initialized && kDebugMode) {
      print('[DEBUG] $message');
    }
  }
  
  static void i(String message) {
    if (_initialized) {
      print('[INFO] $message');
    }
  }
  
  static void w(String message) {
    if (_initialized) {
      print('[WARNING] $message');
    }
  }
  
  static void e(String message) {
    if (_initialized) {
      print('[ERROR] $message');
    }
  }
  
  static void v(String message) {
    if (_initialized && kDebugMode) {
      print('[VERBOSE] $message');
    }
  }
}
