package github

var (
	GithubOauth2AuthorizationCodeFlowCredentialName = "Symphony Platform User Github Credentials"

	GithubVendorName      = "github"
	GithubIntegrationName = "Github Integration"

	// test action
	GithubTestActionName = "TestAction"
	GithubTestActionType = "test_action"
	// get pull request action
	GithubPullOpenRequestActionName = "Get Open Pull Requests"
	GithubPullOpenRequestActionType = "github_list_open_pull_request"
	// get pull request by number action
	GithubGetPullRequestByNumberActionName = "Get Pull Request By Number"
	GithubGetPullRequestByNumberActionType = "github_get_pr_by_num"
	// list commits by pr
	GithubListCommitsActionName = "List Commits By PR"
	GithubListCommitsActionType = "github_list_commits_by_pr"
	// get commits by commit sha
	GithubGetCommitsActionName = "Get Commit By Sha"
	GithubGetCommitsActionType = "github_get_commit_by_sha"
	// add review comment
	GithubAddReviewCommentActionName = "Add a review comment"
	GithubAddReviewCommentActionType = "github_add_review_comment"
)
