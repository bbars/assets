CREATE TABLE IF NOT EXISTS asset (
      asset_key char(32) not null primary key
    , btime timestamp not null default current_timestamp
    , mtime timestamp null default null
    , dtime timestamp null default null
    , size bigint not null default 0
    , content_hash char(64) null
    , content_type varchar(512) null default null
    , original_name varchar(512) null default null
    , user_id varchar(32) null default null
    , original_url varchar(4096) null default null
    , deleted boolean not null default false
    , storage_name varchar(32) not null
    , status ENUM /*('pending', 'processing', 'done')*/ NOT NULL DEFAULT 'pending'
    , info varchar(4096) null default null
    , error varchar(512) null default null
);
