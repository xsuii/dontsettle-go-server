select * from offline_message;
insert into offline_message values(100000,100001,138244234,"adjio",1);
truncate table offline_message;

select * from game.user;
insert into game.user(userName, userEmail, userPassword) values("xsuii", "xsuii@test.com", "123");
insert into user(userName, userEmail, userPassword) values("watson", "watson@test.com", "123");
insert into user(userName, userEmail, userPassword) values("a", "a@test.com", "a");
insert into user(userName, userEmail, userPassword) values("1", "1@test.com", "1");
insert into user(userName, userEmail, userPassword) values("2", "2@test.com", "2");
delete from user where userId < 10000000;
alter table user auto_increment = 100000;
truncate table user;

select * from circle;
insert into circle(circleName) value("circle1");
insert into circle(circleName) value("circle2");
delete from circle where circleId < 100000000;
alter table circle auto_increment = 10000;
truncate table circle;

select * from in_circle;
insert into in_circle values(100000,10000);
insert into in_circle values(100001,10000);
insert into in_circle values(100002,10001);
insert into in_circle values(100003,10001);
truncate table in_circle;
drop table if exists in_circle;
repair table in_circle;
flush table in_circle;
describe in_circle;

select * from file_list;

select * from user where userId in (select userId from in_circle where circleId = 2);