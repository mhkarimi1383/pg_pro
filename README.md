# pg_pro

We want to achieve to a great layer over PostgreSQL, with caching, advanced user management and master/replica load balancing.

## Features

- [x] Parsing Query and checking relation info
- [ ] Caching (By understanding the type of the query)
- [ ] User Management
- [ ] LoadBalancing (By query type [read/write])

## Why

First of all, That's funny and I want to learn how PostgreSQL works (as for connections)

As for User management goes, This project is usefull when you want to manage users with a centeral DB (like OpenLDAP, or a centeral `yaml` file) I only found [Squarespace/pgbedrock](https://github.com/Squarespace/pgbedrock) but it's inactive for 3 years... Also I want to manage users without postgresql itself ;)

As for loadbalancing goes that's great to loadbalance connections to servers in a smart way (By understanding what is the query)

For caching I want to make it posible to use any kind of caching for your database, since postgresql caching parameters are so hard to configure them in a good way.
