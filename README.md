# fragevents

Extra-Life Events Stream

## S&C

1) `heroku plugins:install heroku-kafka`
    1) If not already done
2) `heroku labs:enable runtime-dyno-metadata -a <app name>`
3) `heroku kafka:topics:create --app=fragevents-stage events --retention-time=7d --replication-factor=3 --partitions=8`
   1) Max out retention time up to 14d for prod
4) `heroku kafka:topics:create --app=fragevents-stage teams --compaction --retention-time=7d --replication-factor=3 --partitions=8`
   1) Max out retention time up to 14d for prod
5) `heroku kafka:topics:create --app=fragevents-stage participants --compaction --retention-time=7d --replication-factor=3 --partitions=8`
    1) Max out retention time up to 14d for prod
6) `heroku kafka:topics:create --app=fragevents-stage donations --compaction --retention-time=7d --replication-factor=3 --partitions=8`
    1) Max out retention time up to 14d for prod
7) Set config CFG_GROUPCACHE_TOKEN to a random string of alpha-num between 32 and 128 chars
8) 