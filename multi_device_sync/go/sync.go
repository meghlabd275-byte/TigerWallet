// Multi-Device Sync - Real-time sync, encrypted state transfer

package main

import "sync"

type DeviceSync struct {
    mu sync.RWMutex
    devices map[string]*DeviceState
    pending map[string][]*SyncOp
}

type DeviceState struct {
    DeviceID  string
    Wallet   string
    LastSync int64
    Encrypted []byte
}

type SyncOp struct {
    OpType  string
    Key    string
    Value  []byte
    TS     int64
}

func NewDeviceSync() *DeviceSync {
    return &DeviceSync{
        devices: make(map[string]*DeviceState),
        pending: make(map[string][]*SyncOp),
    }
}

func (s *DeviceSync) RegisterDevice(deviceID, wallet string) error {
    s.devices[deviceID] = &DeviceState{
        DeviceID:  deviceID,
        Wallet:  wallet,
        LastSync: 0,
    }
    return nil
}

func (s *DeviceSync) Sync(deviceID string, ops []*SyncOp) error {
    s.pending[deviceID] = append(s.pending[deviceID], ops...)
    return nil
}

func (s *DeviceSync) GetPending(deviceID string) []*SyncOp {
    return s.pending[deviceID]
}