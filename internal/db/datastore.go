package services

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/google/uuid"
    _ "github.com/lib/pq"

    gitclient "github.com/itau-corp/itau-uq3-app-clone-scan-service/internal/git"
)

// Database é um wrapper fino em torno de *sql.DB para facilitar testes (sqlmock).
type Database struct {
    conn *sql.DB
}

func NewDatabase() *Database { return &Database{} }

// Connect abre conexão PostgreSQL usando lib/pq e valida com Ping().
func (d *Database) Connect(host, port, dbName, user, password string) (*sql.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, dbName, user, password)
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("open conn: %w", err)
    }
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    d.conn = db
    return db, nil
}

func (d *Database) Close() error {
    if d.conn == nil {
        return nil
    }
    return d.conn.Close()
}

// SaveFindings recebe result.Results (lista de BranchScanResult), achata os findings,
// deduplica por fingerprint e insere em lote, garantindo idempotência.
func (d *Database) SaveFindings(ctx context.Context, scanID, repoID uuid.UUID, results []gitclient.BranchScanResult) error {
    if len(results) == 0 {
        return nil
    }

    // Estrutura plana para INSERT
    type flat struct {
        BranchName  string
        File        string
        StartLine   int
        RuleID      string
        Fingerprint string
        Date        time.Time
    }
    uniq := make(map[string]flat)

    for _, br := range results {
        for _, f := range br.Findings {
            ts, _ := time.Parse(time.RFC3339, f.Date)
            uniq[f.Fingerprint] = flat{
                BranchName:  f.BranchName, // usa o campo vindo direto do finding
                File:        f.File,
                StartLine:   f.StartLine,
                RuleID:      f.RuleID,
                Fingerprint: f.Fingerprint,
                Date:        ts,
            }
        }
    }

    if len(uniq) == 0 {
        return nil
    }

    tx, err := d.conn.BeginTx(ctx, &sql.TxOptions{})
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }

    const sqlInsert = `INSERT INTO tbuq3003_resu_expc_crdl_exop (
        codigo_resultado_exploracao,
        scan_id,
        repo_id,
        branch_name,
        file,
        start_line,
        rule_id,
        fingerprint,
        date
    ) VALUES (
        $1,$2,$3,$4,$5,$6,$7,$8,$9
    ) ON CONFLICT (repo_id, rule_id, fingerprint) DO NOTHING`

    stmt, err := tx.PrepareContext(ctx, sqlInsert)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("prepare: %w", err)
    }
    defer stmt.Close()

    for _, v := range uniq {
        if _, err := stmt.ExecContext(ctx,
            uuid.New(),
            scanID,
            repoID,
            v.BranchName,
            v.File,
            v.StartLine,
            v.RuleID,
            v.Fingerprint,
            v.Date,
        ); err != nil {
            tx.Rollback()
            return fmt.Errorf("insert: %w", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    return nil
}
