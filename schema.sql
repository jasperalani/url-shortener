create database `url-shortener`;

use `url-shortener`;

-- auto-generated definition
create table `forwarding-table`
(
    short   varchar(7)                           null,
    `long`      varchar(255)                         null,
    creation    datetime default current_timestamp() null,
    deactivated datetime                             null
);