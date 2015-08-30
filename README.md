# Lab Publicizer

This toll will take all of your Flatiron School lab forks and publicize them. After using this tool, employers will look at your github history and see all the amazing work you've done while a student here.

## How It Works

Github's permissions model is a bit limited, so publicizing a private fork is not as simple as just changing the private/public flag to public. If we publicized the repo on `learn-co-students` then it actually *deletes* all of the forks. That's not good. As a forker, you can't go to the settings and hit public either. Currently, this tool follows the steps laid out in the [duplicating a repository](https://help.github.com/articles/duplicating-a-repository/) documentation the GitHub provides. Here are roughly the steps:

 1. Get all your repos
 2. Filter out only the forks from `learn-co-students`
 3. For each repo, bare clone to `/tmp/cloned/`
 4. Create a new GitHub repo with the fork name plus `-public`
 5. Mirror push to new public Github Repo
