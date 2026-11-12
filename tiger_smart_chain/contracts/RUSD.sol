// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

/// @title RUSD - Royal Tiger USD
/// @notice USD-pegged stablecoin issued natively on TigerSmartChain. Every RUSD
///         in circulation is backed 1:1 by a reserve held by the TigerWallet
///         treasury; issuance (mint) and redemption (burn) are restricted to
///         accounts the owner has authorised as minters, so the supply can never
///         be inflated by an arbitrary caller.
///
/// @dev This contract is deployed as a *genesis predeploy* on TigerSmartChain:
///      its runtime bytecode and its initial storage are committed into the
///      genesis block, so RUSD exists at a fixed address from block 0 and every
///      wallet knows the address without a deployment manifest.
///
///      Because the storage is written directly into the genesis state instead
///      of by a constructor, the storage layout below is part of the consensus
///      surface. Reordering, inserting or removing a state variable changes the
///      predeploy storage and therefore the genesis hash. See
///      chain/predeploy.go, which encodes exactly this layout, and
///      chain/predeploy_test.go, which locks the resulting slots down.
contract RUSD {
    // ---------------------------------------------------------------------
    // Storage layout - DO NOT REORDER
    // ---------------------------------------------------------------------

    /// @dev slot 0 - account balances.
    mapping(address => uint256) private _balances;

    /// @dev slot 1 - ERC-20 allowances, owner => spender => amount.
    mapping(address => mapping(address => uint256)) private _allowances;

    /// @dev slot 2 - total RUSD in circulation, in base units.
    uint256 private _totalSupply;

    /// @dev slot 3 - ERC-20 name (short string).
    string private _name;

    /// @dev slot 4 - ERC-20 symbol (short string).
    string private _symbol;

    /// @dev slot 5 - current owner.
    address private _owner;

    /// @dev slot 6 - accounts authorised to mint and burn (treasury, paymaster).
    mapping(address => bool) private _minters;

    /// @dev slot 7 - circuit breaker. When true every transfer is rejected.
    bool private _paused;

    /// @dev slot 8 - accounts frozen by a compliance action.
    mapping(address => bool) private _frozen;

    /// @dev slot 9 - EIP-2612 permit nonces.
    mapping(address => uint256) private _nonces;

    /// @dev slot 10 - pending owner of the two-step ownership transfer.
    address private _pendingOwner;

    // ---------------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------------

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event Mint(address indexed minter, address indexed to, uint256 value);
    event Burn(address indexed burner, address indexed from, uint256 value);
    event MinterUpdated(address indexed account, bool allowed);
    event Paused(address indexed account);
    event Unpaused(address indexed account);
    event Frozen(address indexed account, bool frozen);
    event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ---------------------------------------------------------------------
    // Errors
    // ---------------------------------------------------------------------

    error NotOwner(address caller);
    error NotMinter(address caller);
    error ZeroAddress();
    error ZeroAmount();
    error InsufficientBalance(address account, uint256 balance, uint256 needed);
    error InsufficientAllowance(address owner, address spender, uint256 allowance, uint256 needed);
    error ContractPaused();
    error AccountFrozen(address account);
    error PermitExpired(uint256 deadline);
    error InvalidSignature();
    error NotPendingOwner(address caller);

    // ---------------------------------------------------------------------
    // Modifiers
    // ---------------------------------------------------------------------

    modifier onlyOwner() {
        if (msg.sender != _owner) revert NotOwner(msg.sender);
        _;
    }

    modifier onlyMinter() {
        if (!_minters[msg.sender]) revert NotMinter(msg.sender);
        _;
    }

    modifier whenNotPaused() {
        if (_paused) revert ContractPaused();
        _;
    }

    // ---------------------------------------------------------------------
    // ERC-20 metadata
    // ---------------------------------------------------------------------

    function name() external view returns (string memory) {
        return _name;
    }

    function symbol() external view returns (string memory) {
        return _symbol;
    }

    function decimals() external pure returns (uint8) {
        return 18;
    }

    function totalSupply() external view returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address account) external view returns (uint256) {
        return _balances[account];
    }

    function allowance(address owner_, address spender) external view returns (uint256) {
        return _allowances[owner_][spender];
    }

    // ---------------------------------------------------------------------
    // ERC-20 transfers
    // ---------------------------------------------------------------------

    function transfer(address to, uint256 value) external whenNotPaused returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function approve(address spender, uint256 value) external whenNotPaused returns (bool) {
        _approve(msg.sender, spender, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external whenNotPaused returns (bool) {
        uint256 allowed = _allowances[from][msg.sender];
        if (allowed != type(uint256).max) {
            if (allowed < value) revert InsufficientAllowance(from, msg.sender, allowed, value);
            unchecked {
                _approve(from, msg.sender, allowed - value);
            }
        }
        _transfer(from, to, value);
        return true;
    }

    function increaseAllowance(address spender, uint256 added) external whenNotPaused returns (bool) {
        _approve(msg.sender, spender, _allowances[msg.sender][spender] + added);
        return true;
    }

    function decreaseAllowance(address spender, uint256 subtracted) external whenNotPaused returns (bool) {
        uint256 allowed = _allowances[msg.sender][spender];
        if (allowed < subtracted) revert InsufficientAllowance(msg.sender, spender, allowed, subtracted);
        unchecked {
            _approve(msg.sender, spender, allowed - subtracted);
        }
        return true;
    }

    function _transfer(address from, address to, uint256 value) private {
        if (to == address(0)) revert ZeroAddress();
        if (_frozen[from]) revert AccountFrozen(from);
        if (_frozen[to]) revert AccountFrozen(to);
        uint256 balance = _balances[from];
        if (balance < value) revert InsufficientBalance(from, balance, value);
        unchecked {
            _balances[from] = balance - value;
            _balances[to] += value;
        }
        emit Transfer(from, to, value);
    }

    function _approve(address owner_, address spender, uint256 value) private {
        if (spender == address(0)) revert ZeroAddress();
        _allowances[owner_][spender] = value;
        emit Approval(owner_, spender, value);
    }

    // ---------------------------------------------------------------------
    // Issuance and redemption (treasury-controlled)
    // ---------------------------------------------------------------------

    /// @notice Issues RUSD against a matching reserve deposit. Restricted to
    ///         authorised minters so the peg cannot be diluted by anyone else.
    function mint(address to, uint256 value) external onlyMinter returns (bool) {
        if (to == address(0)) revert ZeroAddress();
        if (value == 0) revert ZeroAmount();
        _totalSupply += value;
        _balances[to] += value;
        emit Mint(msg.sender, to, value);
        emit Transfer(address(0), to, value);
        return true;
    }

    /// @notice Redeems RUSD held by the caller against the reserve.
    function burn(uint256 value) external whenNotPaused returns (bool) {
        _burn(msg.sender, value);
        return true;
    }

    /// @notice Redeems RUSD held by `from` using an allowance, for custodial and
    ///         paymaster driven redemptions.
    function burnFrom(address from, uint256 value) external whenNotPaused returns (bool) {
        uint256 allowed = _allowances[from][msg.sender];
        if (allowed != type(uint256).max) {
            if (allowed < value) revert InsufficientAllowance(from, msg.sender, allowed, value);
            unchecked {
                _approve(from, msg.sender, allowed - value);
            }
        }
        _burn(from, value);
        return true;
    }

    /// @notice Treasury-side redemption of a frozen account during a compliance
    ///         action. Only the owner can call it and only for a frozen account,
    ///         so it can never be used to seize an ordinary holder's balance.
    function seizeFrozen(address from, uint256 value) external onlyOwner returns (bool) {
        if (!_frozen[from]) revert AccountFrozen(from);
        _burn(from, value);
        return true;
    }

    function _burn(address from, uint256 value) private {
        if (value == 0) revert ZeroAmount();
        uint256 balance = _balances[from];
        if (balance < value) revert InsufficientBalance(from, balance, value);
        unchecked {
            _balances[from] = balance - value;
            _totalSupply -= value;
        }
        emit Burn(msg.sender, from, value);
        emit Transfer(from, address(0), value);
    }

    // ---------------------------------------------------------------------
    // EIP-2612 permit
    // ---------------------------------------------------------------------

    function nonces(address account) external view returns (uint256) {
        return _nonces[account];
    }

    function DOMAIN_SEPARATOR() external view returns (bytes32) {
        return _domainSeparator();
    }

    /// @dev The separator is derived on every call from the live chain id, so a
    ///      signature produced before a chain split can never be replayed on the
    ///      forked chain.
    function _domainSeparator() private view returns (bytes32) {
        return keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256(bytes(_name)),
                keccak256(bytes("1")),
                block.chainid,
                address(this)
            )
        );
    }

    function permit(
        address owner_,
        address spender,
        uint256 value,
        uint256 deadline,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) external whenNotPaused {
        if (block.timestamp > deadline) revert PermitExpired(deadline);
        if (v != 27 && v != 28) revert InvalidSignature();
        // Reject the upper half of the secp256k1 order so a signature cannot be
        // flipped into an equally valid second form (EIP-2 malleability).
        if (uint256(s) > 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0) {
            revert InvalidSignature();
        }

        uint256 nonce = _nonces[owner_];
        bytes32 structHash = keccak256(
            abi.encode(
                keccak256("Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"),
                owner_,
                spender,
                value,
                nonce,
                deadline
            )
        );
        bytes32 digest = keccak256(abi.encodePacked("\x19\x01", _domainSeparator(), structHash));
        address recovered = ecrecover(digest, v, r, s);
        if (recovered == address(0) || recovered != owner_) revert InvalidSignature();

        _nonces[owner_] = nonce + 1;
        _approve(owner_, spender, value);
    }

    // ---------------------------------------------------------------------
    // Administration
    // ---------------------------------------------------------------------

    function owner() external view returns (address) {
        return _owner;
    }

    function pendingOwner() external view returns (address) {
        return _pendingOwner;
    }

    function paused() external view returns (bool) {
        return _paused;
    }

    function isMinter(address account) external view returns (bool) {
        return _minters[account];
    }

    function isFrozen(address account) external view returns (bool) {
        return _frozen[account];
    }

    function addMinter(address account) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        _minters[account] = true;
        emit MinterUpdated(account, true);
    }

    function removeMinter(address account) external onlyOwner {
        _minters[account] = false;
        emit MinterUpdated(account, false);
    }

    function pause() external onlyOwner {
        _paused = true;
        emit Paused(msg.sender);
    }

    function unpause() external onlyOwner {
        _paused = false;
        emit Unpaused(msg.sender);
    }

    function setFrozen(address account, bool frozen_) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        _frozen[account] = frozen_;
        emit Frozen(account, frozen_);
    }

    /// @notice First step of a two-step ownership transfer. The new owner must
    ///         call acceptOwnership, so a typo can never hand the contract to an
    ///         address nobody controls.
    function transferOwnership(address newOwner) external onlyOwner {
        if (newOwner == address(0)) revert ZeroAddress();
        _pendingOwner = newOwner;
        emit OwnershipTransferStarted(_owner, newOwner);
    }

    function acceptOwnership() external {
        if (msg.sender != _pendingOwner) revert NotPendingOwner(msg.sender);
        address previous = _owner;
        _owner = _pendingOwner;
        _pendingOwner = address(0);
        emit OwnershipTransferred(previous, msg.sender);
    }

    function renounceOwnership() external onlyOwner {
        address previous = _owner;
        _owner = address(0);
        _pendingOwner = address(0);
        emit OwnershipTransferred(previous, address(0));
    }
}
