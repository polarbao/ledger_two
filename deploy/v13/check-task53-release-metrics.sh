#!/bin/sh

set -eu

database="${DB_PATH:?DB_PATH must point to an isolated Task53 schema 22 database}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
minimum_sample="${MINIMUM_LEARNED_SAMPLE:-10}"

if [ "$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")" != "22" ]; then
  echo "Task53 release metrics require schema 22" >&2
  exit 1
fi

counts="$("$sqlite_bin" -readonly -separator '|' "$database" "
  WITH learned_matches AS (
    SELECT DISTINCT i.id, i.classification_status,
           COALESCE(i.selected_category_id, '') AS selected_category_id,
           COALESCE(i.suggested_category_id, '') AS suggested_category_id,
           COALESCE(i.selected_tag_ids_json, '[]') AS selected_tags,
           COALESCE(i.suggested_tag_ids_json, '[]') AS suggested_tags
    FROM import_items i
    JOIN import_batches b ON b.id = i.batch_id AND b.status = 'committed'
    JOIN json_each(COALESCE(i.matched_rule_ids_json, '[]')) matched
    JOIN import_rules r ON r.id = matched.value AND r.ledger_id = b.ledger_id
      AND r.origin = 'learned' AND r.apply_mode = 'auto' AND r.confidence = 'high'
    WHERE i.status = 'imported'
  )
  SELECT COUNT(*),
         COALESCE(SUM(CASE WHEN classification_status IN ('manual', 'bulk')
           AND (selected_category_id <> suggested_category_id
             OR EXISTS (
               SELECT value FROM json_each(selected_tags)
               EXCEPT SELECT value FROM json_each(suggested_tags)
             )
             OR EXISTS (
               SELECT value FROM json_each(suggested_tags)
               EXCEPT SELECT value FROM json_each(selected_tags)
             ))
           THEN 1 ELSE 0 END), 0)
  FROM learned_matches;
")"
sample_count="${counts%%|*}"
correction_count="${counts##*|}"
rate="$(awk -v corrected="$correction_count" -v total="$sample_count" 'BEGIN { if (total == 0) print "0.00"; else printf "%.2f", corrected * 100 / total }')"

echo "learned_match_sample=$sample_count"
echo "learned_auto_corrections=$correction_count"
echo "auto_correction_rate_percent=$rate"
if [ "$sample_count" -lt "$minimum_sample" ]; then
  echo "insufficient learned-rule sample; keep IMPORT_CLASSIFICATION_MODE=suggest" >&2
  exit 2
fi
if awk -v value="$rate" 'BEGIN { exit !(value >= 10) }'; then
  echo "auto correction rate reached 10%; learned auto must be downgraded to suggest" >&2
  exit 3
fi
echo "Task53 learned auto correction gate passed"
