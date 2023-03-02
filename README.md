# pg_pro

![banner](./assets/banner.jpg)

We want to achieve to a great layer over PostgreSQL, with caching, advanced user management and master/replica load balancing.

## Features

- [x] Parsing Query and checking relation info
- [x] Caching (By understanding the type of the query)
- [x] User Management (only yaml provider)
- [x] LoadBalancing (By query type [read/write])
- [ ] Automatic master/slave detection

## Why

First of all, That's funny and I want to learn how PostgreSQL works (as for connections)

As for User management goes, This project is usefull when you want to manage users with a centeral DB (like OpenLDAP, or a centeral `yaml` file) I only found [Squarespace/pgbedrock](https://github.com/Squarespace/pgbedrock) but it's inactive for 3 years... Also I want to manage users without postgresql itself ;)

As for loadbalancing goes that's great to loadbalance connections to servers in a smart way (By understanding what is the query)

For caching I want to make it posible to use any kind of caching for your database, since postgresql caching parameters are so hard to configure them in a good way.

## How loadbalancer works

When project starts it will make a connection pool to each server that is listed in the `config.yaml` file

when a quey comes, it will parse given query, then it will return an error to user (if there is any error within the query)

we will select one of the servers to loadbalance between them

## Configuration

Checkout `config.yaml` file this file could be in one of theas directories:

- /etc/pg_pro/
- $HOME/.pg_pro
- $PWD

## Building/Local Development

a Working go environment

you have to set `CGO_ENABLED=1`, since we are using [pganalyze/pg_query_go](https://github.com/pganalyze/pg_query_go) to parse queries

I would recommend to use [cosmtrek/air](https://github.com/cosmtrek/air) for hot reloading and having up-to-data `go build` options

or you can just use `docker` to build/deploy the project
