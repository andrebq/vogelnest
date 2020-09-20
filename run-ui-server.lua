local exec = require('exec')
local fp = require('filepath')
local env = require('env')

env.set('API_ROOT', 'http://localhost:8080')


exec.pushd(fp.join('internal', 'ui'))
exec.execute('yarn', 'install')
exec.execute('yarn', 'run', 'dev')
exec.popd()
