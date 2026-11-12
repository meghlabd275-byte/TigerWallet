// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;

/// @title WTGR - Wrapped TGR
/// @notice ERC-20 wrapper for the native TigerSmartChain coin (TGR). TigerSwap
///         pools are x*y=k pairs over ERC-20 balances, so the native coin needs
///         an ERC-20 representation to be tradable against RUSD. WTGR is that
///         representation and is fully collateralised: every WTGR in existence
///         is backed by exactly one TGR held by this contract.
///
/// @dev Deployed as a genesis predeploy, exactly like RUSD. See
///      chain/predeploy.go for the storage layout.
///
///      deposit() credits WTGR for the TGR sent with the call, so the contract
///      balance always covers totalSupply(). withdraw() burns the WTGR first and
///      only then releases the native coin (checks-effects-interactions), so a
///      re-entrant recipient cannot mint unbacked WTGR.
contract WTGR {
    // ---------------------------------------------------------------------
    // Storage layout - DO NOT REORDER
    // ---------------------------------------------------------------------

    /// @dev slot 0 - ERC-20 name.
    string private _name;

    /// @dev slot 1 - ERC-20 symbol.
    string private _symbol;

    /// @dev slot 2 - wrapped balances.
    mapping(address => uint256) private _balances;

    /// @dev slot 3 - allowances.
    mapping(address => mapping(address => uint256)) private _allowances;

    /// @dev slot 4 - total TGR held by this contract.
    uint256 private _totalSupply;

    /// @dev slot 5 - owner.
    address private _owner;

    /// @dev slot 6 - reentrancy guard.
    uint256 private _locked;

    // ---------------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------------

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event Deposit(address indexed account, uint256 value);
    event Withdrawal(address indexed account, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // ---------------------------------------------------------------------
    // Errors
    // ---------------------------------------------------------------------

    error NotOwner(address caller);
    error ZeroAddress();
    error ZeroAmount();
    error InsufficientBalance(address account, uint256 balance, uint256 needed);
    error InsufficientAllowance(address owner, address spender, uint256 allowance, uint256 needed);
    error NativeTransferFailed(address recipient, uint256 value);
    error Reentrancy();

    modifier onlyOwner() {
        if (msg.sender != _owner) revert NotOwner(msg.sender);
        _;
    }

    modifier nonReentrant() {
        if (_locked == 1) revert Reentrancy();
        _locked = 1;
        _;
        _locked = 0;
    }

    // ---------------------------------------------------------------------
    // Metadata
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

    function owner() external view returns (address) {
        return _owner;
    }

    // ---------------------------------------------------------------------
    // Wrap / unwrap
    // ---------------------------------------------------------------------

    /// @notice Wraps the TGR sent with the call into WTGR credited to the caller.
    function deposit() external payable {
        if (msg.value == 0) revert ZeroAmount();
        _totalSupply += msg.value;
        _balances[msg.sender] += msg.value;
        emit Deposit(msg.sender, msg.value);
        emit Transfer(address(0), msg.sender, msg.value);
    }

    /// @notice Unwraps `value` WTGR back into native TGR for the caller.
    function withdraw(uint256 value) external nonReentrant {
        if (value == 0) revert ZeroAmount();
        uint256 balance = _balances[msg.sender];
        if (balance < value) revert InsufficientBalance(msg.sender, balance, value);

        unchecked {
            _balances[msg.sender] = balance - value;
            _totalSupply -= value;
        }
        emit Transfer(msg.sender, address(0), value);

        (bool ok, ) = msg.sender.call{value: value}("");
        if (!ok) revert NativeTransferFailed(msg.sender, value);
        emit Withdrawal(msg.sender, value);
    }

    // ---------------------------------------------------------------------
    // ERC-20 transfers
    // ---------------------------------------------------------------------

    function transfer(address to, uint256 value) external returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function approve(address spender, uint256 value) external returns (bool) {
        _approve(msg.sender, spender, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external returns (bool) {
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

    function _transfer(address from, address to, uint256 value) private {
        if (to == address(0)) revert ZeroAddress();
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
    // Administration
    // ---------------------------------------------------------------------

    function transferOwnership(address newOwner) external onlyOwner {
        if (newOwner == address(0)) revert ZeroAddress();
        address previous = _owner;
        _owner = newOwner;
        emit OwnershipTransferred(previous, newOwner);
    }

    /// @dev Accepts the TGR sent by deposit(), and the TGR the node pays back to
    ///      this contract while a withdrawal is being executed.
    receive() external payable {}
}
