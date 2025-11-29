#!/bin/bash
cd /var/www/html/mineba-api
go build -o wangov
pm2 start wangov --name mineba-api