// SPDX-License-Identifier: GPL-3.0-or-later

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func (p *Postgres) collect() (map[string]int64, error) {
	if p.db == nil {
		if err := p.openConnection(); err != nil {
			return nil, err
		}
	}

	if p.serverVersion == 0 {
		ver, err := p.queryServerVersion()
		if err != nil {
			return nil, fmt.Errorf("querying server version error: %v", err)
		}
		p.serverVersion = ver
	}

	now := time.Now()

	if now.Sub(p.recheckSettingsTime) > p.recheckSettingsEvery {
		p.recheckSettingsTime = now
		maxConn, err := p.querySettingsMaxConnections()
		if err != nil {
			return nil, fmt.Errorf("querying settings max connections error: %v", err)
		}
		p.maxConnections = maxConn
	}

	if now.Sub(p.relistDatabaseTime) > p.relistDatabaseEvery {
		p.relistDatabaseTime = now
		dbs, err := p.queryDatabaseList()
		if err != nil {
			return nil, fmt.Errorf("querying database list error: %v", err)
		}
		p.collectDatabaseList(dbs)
	}

	mx := make(map[string]int64)

	if err := p.collectConnection(mx); err != nil {
		return mx, fmt.Errorf("querying server connections error: %v", err)
	}

	if err := p.collectCheckpoints(mx); err != nil {
		return mx, fmt.Errorf("querying database conflicts error: %v", err)
	}

	if err := p.collectDatabaseStats(mx); err != nil {
		return mx, fmt.Errorf("querying database stats error: %v", err)
	}

	// TODO: This view will only contain information on standby servers, since conflicts do not occur on primary servers.
	// see if possible to identify primary/standby and disable on primary if yes.
	if err := p.collectDatabaseConflicts(mx); err != nil {
		return mx, fmt.Errorf("querying database conflicts error: %v", err)
	}

	if err := p.collectDatabaseLocks(mx); err != nil {
		return mx, fmt.Errorf("querying database locks error: %v", err)
	}

	return mx, nil
}

func (p *Postgres) openConnection() error {
	db, err := sql.Open("pgx", p.DSN)
	if err != nil {
		return fmt.Errorf("error on opening a connection with the Postgres database [%s]: %v", p.DSN, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout.Duration)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("error on pinging the Postgres database [%s]: %v", p.DSN, err)
	}
	p.db = db

	return nil
}

func (p *Postgres) querySettingsMaxConnections() (int64, error) {
	q := querySettingsMaxConnections()

	var v string
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout.Duration)
	defer cancel()
	if err := p.db.QueryRowContext(ctx, q).Scan(&v); err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func (p *Postgres) queryServerVersion() (int, error) {
	q := queryServerVersion()

	var v string
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout.Duration)
	defer cancel()
	if err := p.db.QueryRowContext(ctx, q).Scan(&v); err != nil {
		return 0, err
	}
	return strconv.Atoi(v)
}

//func (p *Postgres) queryIsSuperUser() (bool, error) {
//	q := queryIsSuperUser()
//
//	var v bool
//	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout.Duration)
//	defer cancel()
//	if err := p.db.QueryRowContext(ctx, q).Scan(&v); err != nil {
//		return false, err
//	}
//	return v, nil
//}

func collectRows(rows *sql.Rows, assign func(column, value string)) error {
	if assign == nil {
		return nil
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := makeNullStrings(len(columns))

	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return err
		}
		for i, v := range values {
			assign(columns[i], valueToString(v))
		}
	}
	return nil
}

func valueToString(value interface{}) string {
	v, ok := value.(*sql.NullString)
	if !ok || !v.Valid {
		return ""
	}
	return v.String
}

func makeNullStrings(size int) []interface{} {
	vs := make([]interface{}, size)
	for i := range vs {
		vs[i] = &sql.NullString{}
	}
	return vs
}

func safeParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func calcPercentage(value, total int64) int64 {
	if total == 0 {
		return 0
	}
	return value * 100 / total
}
