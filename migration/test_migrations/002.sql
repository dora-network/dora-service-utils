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
-- +start
insert into addresses (id, address1, address2, address3, city, postcode)
values (100, '100 Some Street', 'Some place', 'Somewhere', 'London', 'N1 2AB')
-- +end
