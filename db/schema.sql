create table if not exists links (
                                     id uuid primary key,
                                     original_url text not null,
                                     short_code varchar(50) unique not null, -- короткий хеш или кастомное имя
                                     created timestamp with time zone default now()
);

create index idx_short_code on links(short_code);

create table if not exists clicks (
                                      id uuid primary key,
                                      link_id uuid not null references links(id) on delete cascade,
                                      user_agent text,
                                      ip_address varchar(45),
                                      clicked timestamp with time zone default now()
);

create index idx_clicked_at on clicks(clicked);
