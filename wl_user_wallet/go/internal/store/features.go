// Real PostgreSQL persistence for the flat-route feature entities:
// price alerts, P2P, DAO, launchpool, token sales, token approvals, fees,
// KYC records, cards, margin & perpetual positions.
package store

import (
        "context"
        "errors"

        "github.com/google/uuid"
        "github.com/jackc/pgx/v5"
)

type PriceAlert struct {
        ID        string
        Symbol    string
        Target    string
        Direction string
        Enabled   bool
}

func (s *Store) ListPriceAlerts(ctx context.Context, userID uuid.UUID) ([]PriceAlert, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, symbol, target_price, direction, enabled FROM price_alerts WHERE user_id=$1 ORDER BY created_at DESC`, userID)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []PriceAlert{}
        for rows.Next() {
                var a PriceAlert
                if err := rows.Scan(&a.ID, &a.Symbol, &a.Target, &a.Direction, &a.Enabled); err != nil { continue }
                out = append(out, a)
        }
        return out, nil
}

func (s *Store) CreatePriceAlert(ctx context.Context, userID uuid.UUID, symbol, target, direction string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO price_alerts (id, user_id, symbol, target_price, direction) VALUES ($1,$2,$3,$4::numeric,$5)`,
                id, userID, symbol, target, direction)
        return id, err
}

func (s *Store) DeletePriceAlert(ctx context.Context, userID uuid.UUID, id uuid.UUID) (int64, error) {
        tag, err := s.db.Exec(ctx, `DELETE FROM price_alerts WHERE id=$1 AND user_id=$2`, id, userID)
        if err != nil { return 0, err }
        return tag.RowsAffected(), nil
}

// ---- P2P ----
type P2PAdvert struct {
        ID    string
        Side  string
        Asset string
        Price string
}

func (s *Store) ListP2PAdverts(ctx context.Context) ([]P2PAdvert, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, side, asset, price FROM p2p_adverts ORDER BY created_at DESC`)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []P2PAdvert{}
        for rows.Next() {
                var a P2PAdvert
                if err := rows.Scan(&a.ID, &a.Side, &a.Asset, &a.Price); err != nil { continue }
                out = append(out, a)
        }
        return out, nil
}

func (s *Store) CreateP2PAdvert(ctx context.Context, userID uuid.UUID, side, asset, price string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO p2p_adverts (id, user_id, side, asset, price) VALUES ($1,$2,$3,$4,$5::numeric)`,
                id, userID, side, asset, price)
        return id, err
}

func (s *Store) CreateP2POrder(ctx context.Context, advertID, buyerID uuid.UUID, amount string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO p2p_orders (id, advert_id, buyer_id, amount) VALUES ($1,$2,$3,$4::numeric)`,
                id, advertID, buyerID, amount)
        return id, err
}

// ---- DAO ----
type DaoProposal struct {
        ID           string
        Title        string
        Description  string
        VotesFor     int64
        VotesAgainst int64
}

func (s *Store) ListDaoProposals(ctx context.Context) ([]DaoProposal, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, title, description, votes_for, votes_against FROM dao_proposals ORDER BY created_at DESC`)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []DaoProposal{}
        for rows.Next() {
                var p DaoProposal
                if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.VotesFor, &p.VotesAgainst); err != nil { continue }
                out = append(out, p)
        }
        return out, nil
}

func (s *Store) CreateDaoProposal(ctx context.Context, title, description string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO dao_proposals (id, title, description) VALUES ($1,$2,$3)`,
                id, title, description)
        return id, err
}

func (s *Store) VoteDaoProposal(ctx context.Context, proposalID, voterID uuid.UUID, support bool) error {
        _, err := s.db.Exec(ctx,
                `INSERT INTO dao_votes (proposal_id, voter_id, support) VALUES ($1,$2,$3)`,
                proposalID, voterID, support)
        return err
}

// ---- Launchpool ----
type LaunchpoolStake struct {
        ID     string
        Amount string
}

func (s *Store) ListLaunchpoolStakes(ctx context.Context, userID uuid.UUID) ([]LaunchpoolStake, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, amount FROM launchpool WHERE user_id=$1 ORDER BY created_at DESC`, userID)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []LaunchpoolStake{}
        for rows.Next() {
                var ls LaunchpoolStake
                if err := rows.Scan(&ls.ID, &ls.Amount); err != nil { continue }
                out = append(out, ls)
        }
        return out, nil
}

func (s *Store) LaunchpoolStake(ctx context.Context, userID uuid.UUID, amount string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO launchpool (id, user_id, amount) VALUES ($1,$2,$3::numeric)`, id, userID, amount)
        return id, err
}

// ---- Token sales ----
type TokenSale struct {
        ID     string
        Name   string
        Symbol string
        Supply string
        Raised string
}

func (s *Store) ListTokenSales(ctx context.Context) ([]TokenSale, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, name, symbol, supply, raised FROM token_sales ORDER BY created_at DESC`)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []TokenSale{}
        for rows.Next() {
                var ts TokenSale
                if err := rows.Scan(&ts.ID, &ts.Name, &ts.Symbol, &ts.Supply, &ts.Raised); err != nil { continue }
                out = append(out, ts)
        }
        return out, nil
}

func (s *Store) ParticipateTokenSale(ctx context.Context, saleID, userID uuid.UUID, amount string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO token_sale_entry (id, sale_id, user_id, amount) VALUES ($1,$2,$3,$4::numeric)`,
                id, saleID, userID, amount)
        return id, err
}

// ---- Token approvals ----
type TokenApproval struct {
        ID       string
        Token    string
        Spender  string
        Allowance string
}

func (s *Store) ListTokenApprovals(ctx context.Context, userID uuid.UUID) ([]TokenApproval, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, token, spender, allowance FROM token_approvals WHERE user_id=$1 ORDER BY created_at DESC`, userID)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []TokenApproval{}
        for rows.Next() {
                var a TokenApproval
                if err := rows.Scan(&a.ID, &a.Token, &a.Spender, &a.Allowance); err != nil { continue }
                out = append(out, a)
        }
        return out, nil
}

func (s *Store) RevokeTokenApproval(ctx context.Context, userID uuid.UUID, id uuid.UUID) (int64, error) {
        tag, err := s.db.Exec(ctx, `DELETE FROM token_approvals WHERE id=$1 AND user_id=$2`, id, userID)
        if err != nil { return 0, err }
        return tag.RowsAffected(), nil
}

// ---- Fees ----
type Fee struct {
        ID      string
        WalletID string
        FeeType string
        Amount  string
        TxHash  string
}

func (s *Store) ListFees(ctx context.Context) ([]Fee, error) {
        rows, err := s.db.Query(ctx,
                `SELECT id, wallet_id, fee_type, amount, tx_hash FROM fees ORDER BY created_at DESC`)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []Fee{}
        for rows.Next() {
                var f Fee
                if err := rows.Scan(&f.ID, &f.WalletID, &f.FeeType, &f.Amount, &f.TxHash); err != nil { continue }
                out = append(out, f)
        }
        return out, nil
}

func (s *Store) FeeRevenue(ctx context.Context) (string, error) {
        var total string
        err := s.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM fees`).Scan(&total)
        if err != nil { return "", err }
        return total, nil
}



// ---- KYC ----
type KycRecord struct {
        ID       string
        Status   string
        FullName string
        DocType  string
        DocNum   string
}

func (s *Store) KycRecordFor(ctx context.Context, userID uuid.UUID) (*KycRecord, error) {
        var k KycRecord
        err := s.db.QueryRow(ctx,
                `SELECT id, status, full_name, doc_type, doc_number FROM kyc_records WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`,
                userID).Scan(&k.ID, &k.Status, &k.FullName, &k.DocType, &k.DocNum)
        if err != nil { return nil, err }
        return &k, nil
}

func (s *Store) UpsertKycRecord(ctx context.Context, userID uuid.UUID, fullName, docType, docNumber string) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO kyc_records (id, user_id, status, full_name, doc_type, doc_number) VALUES ($1,$2,'pending',$3,$4,$5)`,
                id, userID, fullName, docType, docNumber)
        return id, err
}

// ---- Cards ----
type CardTx struct {
        ID       string
        Amount   string
        Merchant string
        Status   string
}

type CardAccount struct {
        ID      string
        Balance string
}

func (s *Store) CardAccount(ctx context.Context, userID uuid.UUID) (*CardAccount, error) {
        var ca CardAccount
        err := s.db.QueryRow(ctx,
                `SELECT id, balance FROM card_accounts WHERE user_id=$1`, userID).Scan(&ca.ID, &ca.Balance)
        if err != nil {
                if pgNoRows(err) {
                        // Auto-create with zero balance
                        id := uuid.NewString()
                        _, _ = s.db.Exec(ctx,
                                `INSERT INTO card_accounts (id, user_id, balance) VALUES ($1,$2,0)`, id, userID)
                        return &CardAccount{ID: id, Balance: "0"}, nil
                }
                return nil, err
        }
        return &ca, nil
}

func (s *Store) ListCardTransactions(ctx context.Context, userID uuid.UUID) ([]CardTx, error) {
        var out []CardTx
        rows, err := s.db.Query(ctx,
                `SELECT id, amount, merchant, status FROM card_transactions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
        if err != nil { return nil, err }
        defer rows.Close()
        for rows.Next() {
                var tx CardTx
                if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Merchant, &tx.Status); err != nil { continue }
                out = append(out, tx)
        }
        return out, nil
}

func pgNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// ---- Margin/Perp ----
type Position struct {
        ID       string
        Pair     string
        Side     string
        Size     string
        Leverage int
        Pnl      string
        Status   string
}

func (s *Store) ListMarginPositions(ctx context.Context, userID uuid.UUID) ([]Position, error) {
        return listPositions(ctx, s, userID, "margin_positions")
}

func (s *Store) ListPerpPositions(ctx context.Context, userID uuid.UUID) ([]Position, error) {
        return listPositions(ctx, s, userID, "perp_positions")
}

func listPositions(ctx context.Context, s *Store, userID uuid.UUID, table string) ([]Position, error) {
        query := `SELECT id, pair, side, size, leverage, pnl, status FROM ` + table +
                ` WHERE user_id=$1 ORDER BY created_at DESC`
        rows, err := s.db.Query(ctx, query, userID)
        if err != nil { return nil, err }
        defer rows.Close()
        out := []Position{}
        for rows.Next() {
                var p Position
                if err := rows.Scan(&p.ID, &p.Pair, &p.Side, &p.Size, &p.Leverage, &p.Pnl, &p.Status); err != nil { continue }
                out = append(out, p)
        }
        return out, nil
}

func (s *Store) CreateMarginPosition(ctx context.Context, userID uuid.UUID, pair, side, size string, leverage int) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO margin_positions (id, user_id, pair, side, size, leverage, pnl, status) VALUES ($1,$2,$3,$4,$5::numeric,$6,0,'open')`,
                id, userID, pair, side, size, leverage)
        return id, err
}

func (s *Store) CreatePerpPosition(ctx context.Context, userID uuid.UUID, pair, side, size string, leverage int) (string, error) {
        id := uuid.NewString()
        _, err := s.db.Exec(ctx,
                `INSERT INTO perp_positions (id, user_id, pair, side, size, leverage, entry, pnl, status) VALUES ($1,$2,$3,$4,$5::numeric,$6,0,0,'open')`,
                id, userID, pair, side, size, leverage)
        return id, err
}

func (s *Store) ClosePosition(ctx context.Context, userID uuid.UUID, table string, id uuid.UUID, pnl string) (int64, error) {
        tag, err := s.db.Exec(ctx,
                `UPDATE `+table+` SET status='closed', pnl=$3::numeric WHERE id=$1 AND user_id=$2`, id, userID, pnl)
        if err != nil { return 0, err }
        return tag.RowsAffected(), nil
}
