package achievements

import (
	"brochievements/internal/storage"
	"database/sql"
	"time"
)

func LoadWeeklyDoves(store *storage.Storage) (*DovePair, error) {
	from := time.Now().AddDate(0, 0, -7)

	row := store.DB.QueryRow(`
WITH sessions AS (
	SELECT
		vcs.user_id,
		vcs.channel_id,
		vcs.joined_at,
		vcs.left_at
	FROM voice_channel_sessions vcs
	WHERE vcs.joined_at >= ?
	  AND vcs.left_at IS NOT NULL
),
pairs AS (
	SELECT
		a.user_id AS user_a,
		b.user_id AS user_b,
		a.channel_id,
		MAX(a.joined_at, b.joined_at) AS overlap_start,
		MIN(a.left_at, b.left_at)     AS overlap_end
	FROM sessions a
	JOIN sessions b
		ON a.channel_id = b.channel_id
	   AND a.user_id < b.user_id
	   AND a.joined_at < b.left_at
	   AND b.joined_at < a.left_at
),
valid_pairs AS (
	SELECT
		user_a,
		user_b,
		channel_id,
		(strftime('%s', overlap_end) - strftime('%s', overlap_start)) AS seconds,
		overlap_start,
		overlap_end
	FROM pairs
	WHERE overlap_end > overlap_start
),
only_two AS (
	SELECT vp.*
	FROM valid_pairs vp
	WHERE (
		SELECT COUNT(DISTINCT s.user_id)
		FROM sessions s
		WHERE s.channel_id = vp.channel_id
		  AND s.joined_at < vp.overlap_end
		  AND s.left_at > vp.overlap_start
	) = 2
)
SELECT
	o.user_a,
	o.user_b,
	ua.username AS name_a,
	ub.username AS name_b,
	SUM(o.seconds) AS total_seconds
FROM only_two o
JOIN users ua ON ua.user_id = o.user_a
JOIN users ub ON ub.user_id = o.user_b
GROUP BY o.user_a, o.user_b
ORDER BY total_seconds DESC
LIMIT 1;
`, from)

	var pair DovePair
	err := row.Scan(
		&pair.UserA,
		&pair.UserB,
		&pair.NameA,
		&pair.NameB,
		&pair.Seconds,
	)
	if err == sql.ErrNoRows {
		// это НОРМАЛЬНО — просто нет голубков на этой неделе
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// порог: минимум 1 час
	if pair.Seconds < 3600 {
		return nil, nil
	}

	return &pair, nil
}
