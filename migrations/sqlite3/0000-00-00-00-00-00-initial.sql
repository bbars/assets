CREATE TABLE IF NOT EXISTS assets_migration (
      name varchar(512) not null
    , btime timestamp not null default current_timestamp
    , error varchar(512) not null
);
