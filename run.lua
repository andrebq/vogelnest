local exec = require('exec')
require('./secrets')

local ok, _exitCode, err = exec.execute("go", "build", ".")
if not ok then
    print(err)
    return 0
end

exec.execute("./vogelnest", "-terms", "@jairbolsonaro,pantanal,#pantanal,#queimada")
