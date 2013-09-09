#!/usr/bin/env bash


mysql -u root -e "drop database cp; create database cp;"
mysql -u root cp -e "source svc/database.sql"

