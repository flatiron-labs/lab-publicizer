package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/howeyc/gopass"
	"github.com/libgit2/git2go"
	"golang.org/x/oauth2"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

var client *github.Client
var passphrase string

const CLONE_PATH = "/tmp/cloned"

func main() {
	err := checkPubPrivSSHKeyExists()
	if err != nil {
		panic(err)
	}

	askForSSHPassphrase()
	accessToken := askForGithubAccessToken()

	client = githubClient(accessToken)

	allRepos := getAllRepos()
	forks := filterOnlyForks(allRepos)
	studentForks := filterOnlyStudentRepos(forks)
	duplicateRepositories(studentForks)

}

func checkPubPrivSSHKeyExists() error {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	publicKey := usr.HomeDir + "/.ssh/id_rsa.pub"
	privateKey := usr.HomeDir + "/.ssh/id_rsa"

	if _, err := os.Stat(publicKey); os.IsNotExist(err) {
		return fmt.Errorf("Couldn't find your public key. Looking in: %v", publicKey)
	}
	if _, err := os.Stat(usr.HomeDir + "/.ssh/id_rsa"); os.IsNotExist(err) {
		return fmt.Errorf("Couldn't find your private key. Looking in: %v", privateKey)
	}
	return nil
}

func askForSSHPassphrase() {
	fmt.Print("Enter your ssh passphrase: ")
	passphrase = string(gopass.GetPasswd())
	passphrase = strings.TrimSpace(passphrase)
}

func askForGithubAccessToken() string {
	accessToken := ""

	for accessToken == "" {
		fmt.Println("Head over to https://github.com/settings/tokens and click Generate Token")
		fmt.Print("Paste the token that is created here:")
		fmt.Scanf("%s", &accessToken)
		accessToken = strings.TrimSpace(accessToken)
	}
	return accessToken
}

func githubClient(accessToken string) *github.Client {
	oauthToken := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)

	tc := oauth2.NewClient(oauth2.NoContext, oauthToken)
	return github.NewClient(tc)
}

func getAllRepos() []github.Repository {
	options := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "owner",
	}

	var allRepos []github.Repository
	fmt.Println("Getting all your repos")
	currentPage := 1
	for {
		repos, resp, err := client.Repositories.List("", options)

		fmt.Printf("Downloading Page %v of %v\n", currentPage, resp.LastPage+1)
		currentPage++

		if err != nil {
			errorMessage := fmt.Sprintf("Had trouble receiving repos: %v", err)
			panic(errorMessage)
		}

		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}

		options.ListOptions.Page = resp.NextPage
	}

	fmt.Println("Finished downloading all your repos")
	return allRepos
}

func filterOnlyForks(allRepos []github.Repository) []github.Repository {
	fmt.Println("Finding only your forks")
	forks := allRepos[:0]
	for _, repo := range allRepos {
		if *repo.Fork {
			forks = append(forks, repo)
		}
	}
	fmt.Println("Found all your forks")
	return forks
}

func filterOnlyStudentRepos(resolvedRepos []github.Repository) []github.Repository {
	fmt.Println("Finding just the learn-co-students forks")
	studentForks := resolvedRepos[:0]
	for _, repo := range resolvedRepos {
		fullRepo, _, err := client.Repositories.Get(*repo.Owner.Login, *repo.Name)
		if err == nil && *fullRepo.Parent.Owner.Login == "learn-co-students" {
			studentForks = append(studentForks, repo)
		}

	}

	fmt.Println("Found your learn-co-students forks")

	return studentForks
}

func duplicateRepositories(repos []github.Repository) {
	fmt.Println("Duplicating Repos")
	for _, studentFork := range repos {
		fmt.Printf("Duplicating %v\n", *studentFork.Name)
		duplicateRepository(studentFork)
		// client.Repositories.Delete(*studentFork.Owner.Login, *studentFork.Name+"-public")
	}
	fmt.Println("Finished Duplicating")

	cleanUpCloneDir(CLONE_PATH)

}

func duplicateRepository(repo github.Repository) {
	bareCloneRepo(repo, CLONE_PATH)
	githubRepo, err := createNewPublicRepo(repo)
	if err != nil {
		panic(err)
	}
	err = mirrorPushRepoToGihub(*githubRepo, CLONE_PATH)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func bareCloneRepo(repo github.Repository, clonedPath string) {
	fmt.Printf("Bare Cloning %v\n", *repo.Name)
	cloneOptions := &git.CloneOptions{
		Bare: true,
	}
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      credentialsCallback,
			CertificateCheckCallback: certificateCheckCallback,
		},
	}

	exec.Command("rm", "-Rf", clonedPath).Run()
	_, err := git.Clone(*repo.SSHURL, clonedPath, cloneOptions)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func createNewPublicRepo(repo github.Repository) (*github.Repository, error) {
	newRepo := &github.Repository{
		Name: github.String(*repo.Name + "-public"),
	}
	fmt.Printf("Creating New Public Repo %v\n", *newRepo.Name)
	githubRepo, _, githuberr := client.Repositories.Create("", newRepo)
	if githuberr != nil {
		err := fmt.Errorf("Having trouble print %v: error: %v", *repo.Name, githuberr)
		return nil, err
	}
	return githubRepo, nil

}

func mirrorPushRepoToGihub(repo github.Repository, clonedPath string) error {
	fmt.Printf("Pushing to %v\n", *repo.Name)
	os.Chdir(clonedPath)
	_, giterror := exec.Command("git", "push", "--mirror", *repo.SSHURL).CombinedOutput()
	if giterror != nil {
		return fmt.Errorf("Having trouble pushing repo %v. Error: %v", *repo.Name, giterror)
	}
	return nil

}

func cleanUpCloneDir(clonedPath string) {
	exec.Command("rm", "-Rf", clonedPath).Run()
}

func credentialsCallback(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	ret, cred := git.NewCredSshKey("git", usr.HomeDir+"/.ssh/id_rsa.pub", usr.HomeDir+"/.ssh/id_rsa", passphrase)
	return git.ErrorCode(ret), &cred
}

// Made this one just return 0 during troubleshooting...
func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	return 0
}
