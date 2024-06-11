-- +start

create table addresses (
    id bigint not null primary key,
    address1 varchar not null,
    address2 varchar,
    address3 varchar,
    city varchar,
    postcode varchar
)

-- +end
