-- +Start

-- This is a single line comment

/* This is also a single line comment */

/* This is a multiline comment
   that spans multiple lines
 */

create table users (
    id bigint not null primary key,
    name varchar not null,
    email varchar not null
);

-- +End
