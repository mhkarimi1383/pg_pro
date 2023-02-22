# pg_pro

We want to achieve to a great layer over PostgreSQL, with caching, advanced user management and master/replica load balancing.

## Features

- [x] Parsing Query and checking relation info
- [ ] Caching (By understanding the type of the query)
- [ ] User Management
- [x] LoadBalancing (By query type [read/write])
- [ ] Automatic master/slave detection

## Why

First of all, That's funny and I want to learn how PostgreSQL works (as for connections)

As for User management goes, This project is usefull when you want to manage users with a centeral DB (like OpenLDAP, or a centeral `yaml` file) I only found [Squarespace/pgbedrock](https://github.com/Squarespace/pgbedrock) but it's inactive for 3 years... Also I want to manage users without postgresql itself ;)

As for loadbalancing goes that's great to loadbalance connections to servers in a smart way (By understanding what is the query)

For caching I want to make it posible to use any kind of caching for your database, since postgresql caching parameters are so hard to configure them in a good way.

## How the loadbalancer works

When project starts it will make a connection pool to each server that is listed in the `config.yaml` file

when a quey came, it will parse given query, then it will return an error to user (if there is an error in the query)

when we know the type of the query (read or write), we will select one of the servers to loadbalance between them

## Configuration

Checkout `config.yaml` file this file could be in one of theas directories:

- /etc/pg_pro/
- $HOME/.pg_pro
- $PWD
