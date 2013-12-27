select * from game.group;
select * from game.User;

drop table game.Group;
drop table game.User;

select FROM_UNIXTIME(1156219870);

select * from game.test;

insert into game.test value(9223372036854775807,"1970-01-01","2038-01-19 03:14:07");
insert into game.test value(3,"1970-01-01","2038-01-19 03:14:07");
insert into game.test value(5,"1970-01-01","1156219870");

select * from game.offlinemessage;
truncate game.offlinemessage;