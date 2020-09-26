local exec = require('exec')
local env = require('env')
local fp = require('filepath')
local string = require('string')

local ok = exec.execute("make", "build-image")
if not ok then println("Unable to build image") end
env.set('CORS_ORIGINS', '*.ep.cluster.amoraes.info,*')

require('./secrets')

exec.execute("docker", "run", "--name", "vogelnest", "--rm", "-ti",
    "-v", string.format("%s:%s", fp.abs(fp.join('internal', 'ui', 'dist')), "/opt/vogelnest/static"),
    "-v", string.format("%s:%s", fp.abs(fp.join("testvolume", "tweets")), "/var/data/vogelnest/tweets"),
    "-e", "CORS_ORIGINS",
    "-e", 'TWITTER_API_KEY',
    "-e", 'TWITTER_API_SECRET_KEY',
    "-e", 'TWITTER_ACCESS_TOKEN',
    "-e", 'TWITTER_ACCESS_TOKEN_SECRET',
    "-p", "8080:8080",
    "andrebq/vogelnest:latest")
