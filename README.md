# vogelnest

A twitter client to extract relations between tags, users and words using
graph databases

## How to run?

- Install the [glua](https://github.com/andrebq/glua/) utility as it is used
to provide a consistent environment between windows, linux and macos

- Install go sdk

- Install make

- Compile

- Check the **run** target in **Makefile** and the **run.lua** file

- Add your twitter api secrets in **secrets.lua** (do not commit this file).
You can see an example in **example-secret.lua**

## Why AGLP and not MIT/MPL/Apache?

Most of my code are released under one of those 3 license, but
this one I think is a bit special.

It is special because it will interact with social networks,
I don't think I am the first one to have a similar idea,
but anything that touches social networks has the potential to be
abused to make the world even more disfunctional.

If that happens and people use this tool, they would need to expose
all their source-code otherwise they won't comply with the license.

Having said that, I don't have any means to enforce AGPL compliance,
but others might pickup the fight. If I were to choose other license,
then there wouldn't be a possibility of picking up a fight.
