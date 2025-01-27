// SPDX-License-Identifier: GPL-3.0-or-later

package postgres

import (
	"context"
)

func (p *Postgres) collectCheckpoints(mx map[string]int64) error {
	q := queryCheckpoints()

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout.Duration)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	return collectRows(rows, func(column, value string) { mx[column] = safeParseInt(value) })
}
