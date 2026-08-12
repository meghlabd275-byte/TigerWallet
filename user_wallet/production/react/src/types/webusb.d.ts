/**
 * Minimal WebUSB type declarations.
 *
 * The DOM lib does not include the WebUSB API types, so we declare the subset
 * used by HardwareWalletService here. These match the W3C WebUSB spec
 * (https://wicg.github.io/webusb). HardwareWalletService uses the real
 * `navigator.usb` API for Ledger/Trezor device discovery; these types make
 * that code type-check. No runtime impact.
 */

export {};

declare global {
  interface USBDevice {
    vendorId: number;
    productId: number;
    serialNumber?: string;
    productName?: string;
    manufacturerName?: string;
    opened: boolean;
    configuration: { configurationValue: number } | null;
    open(): Promise<void>;
    close(): Promise<void>;
    selectConfiguration(configurationValue: number): Promise<void>;
    claimInterface(interfaceNumber: number): Promise<void>;
    releaseInterface(interfaceNumber: number): Promise<void>;
    transferIn(
      endpointNumber: number,
      length: number,
    ): Promise<{ data: ArrayBuffer; status: string }>;
    transferOut(
      endpointNumber: number,
      data: BufferSource,
    ): Promise<{ bytesWritten: number; status: string }>;
  }

  interface USBDeviceEventMap {
    connect: USBConnectionEvent;
    disconnect: USBConnectionEvent;
  }

  interface USBConnectionEvent extends Event {
    device: USBDevice;
  }

  interface USB extends EventTarget {
    addEventListener<K extends keyof USBDeviceEventMap>(
      type: K,
      listener: (this: USB, ev: USBDeviceEventMap[K]) => unknown,
    ): void;
    addEventListener(
      type: string,
      listener: EventListenerOrEventListenerObject,
    ): void;
    removeEventListener<K extends keyof USBDeviceEventMap>(
      type: K,
      listener: (this: USB, ev: USBDeviceEventMap[K]) => unknown,
    ): void;
    requestDevice(options: { filters: { vendorId?: number; productId?: number }[] }): Promise<USBDevice>;
    getDevices(): Promise<USBDevice[]>;
  }

  interface Navigator {
    readonly usb?: USB;
  }
}
