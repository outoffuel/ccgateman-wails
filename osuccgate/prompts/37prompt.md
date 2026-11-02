Git管理されているファイルからSQLiteの一時ファイルを除外し、バイナリ実行時や運用時に生成されうる個人情報・一時ファイルが誤ってコミットされないよう `.gitignore` を強化してください。

1. 追跡済みSQLite一時ファイルのインデックス除外
   - 現在Git管理に入ってしまっている `osuccgate/osuccgate.db-shm` と `osuccgate/osuccgate.db-wal` を `git rm --cached` でGitの追跡対象から外してください。

2. `.gitignore` の強化
   以下のパターンを `.gitignore` に追加してください：
   - SQLite関連: `*.db`, `*.db-shm`, `*.db-wal`, `*.db-journal`, `*.sqlite`, `*.sqlite3`
   - 入出力データ・バックアップ（個人情報保護）: `*.csv`, `*.bak`, `*.backup`
   - 自己アップデート・実行バイナリ関連: `*.old`, `*.tmp`, `update-tmp-*`, `osuccgate-update-*`, `*.tar.gz`, `osuccgate/osuccgate`
   - 環境設定ファイル: `.env`, `.env.*`

