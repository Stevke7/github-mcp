package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

var ghClient *github.Client

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required.\n" + "Create a Personal Access Token at: https://github.com/settings/tokens\n" + "It needs theese scopes: repo, delete_repo, workflow")
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)

	ghClient = github.NewClient(tc)

	mcpServer := server.NewMCPServer("github-mcp-server", "1.0.0", server.WithToolCapabilities(false),)

	//Tool 1
	mcpServer.AddTool(
		mcp.NewTool(
			"gitgub_list_repos",
		mcp.WithDescription("List all repositories for the authenticated GitHub user." + 
			"Returns repo name, visibility (public/private), description, and URL."),
		mcp.WithNumber("per_page", mcp.Description("Number of repositories to return per page (1-100, default 30)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination (default 1)")),
		mcp.WithTitleAnnotation("List Repositories"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		), 
		handleListRepos,
	) 
	
	//Tool 2
	mcpServer.AddTool(
		mcp.NewTool(
			"github_create_repo", 
			mcp.WithDescription("Create a new Github repository for authenticated user."),
			mcp.WithString("name", 
		mcp.Required(),
		mcp.Description("Repository name (e.g., 'my-new-project')"),),
		mcp.WithString("description", mcp.Description("Short description of the repository"),),
		mcp.WithBoolean("private", mcp.Description("Whether the repository should be private (default false = public)"),),
		mcp.WithBoolean("auto_init", mcp.Description("Initaialize with a README.md (default false)"),),
		mcp.WithTitleAnnotation("Create Repository"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),     // creating twice = error
		mcp.WithOpenWorldHintAnnotation(true),
		),
		handleCreateRepo,
	)
	
	// --- TOOL 3: Delete Repo ---
	mcpServer.AddTool(
		mcp.NewTool(
			"github_delete_repo",
			mcp.WithDescription("Permanently delete a GitHub repository. "+ "WARNING: This action is irreversible! All code, issues, PRs will be lost."),
			mcp.WithString("Owner", mcp.Required(), mcp.Description("Repository name to delete"),),
			mcp.WithTitleAnnotation("Delete Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),

		), 
		handleDeleteRepo,
	)

	// --- TOOL 4: Update Repo visibility ---
	mcpServer.AddTool(
		mcp.NewTool(
			"github_update_visibility",
			mcp.WithDescription("Change a repository's visibility between public and private"),
			mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner (your Github username)"),),
			mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name"),),
			mcp.WithString("visibility", mcp.Required(), mcp.Description("New visibility: 'public' or 'private'"), mcp.Enum("public", "private"),),
			mcp.WithTitleAnnotation("Update Repository Visibility"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		handleUpdateVisibility,
	)
	
	// --- TOOL 5: Trigger Github Actions Workflow ---
	mcpServer.AddTool(
		mcp.NewTool(
			"github_trigger_workflow", 
			mcp.WithDescription("Trigger a GitHub Actions workflow (CI/CD pipeline) via workflow_dispatch event. "+
				"The workflow must have a 'workflow_dispatch' trigger defined in its YAML file."),
			mcp.WithString("owner", 
			mcp.Required(),
			mcp.Description("Repository owner (your GitHub username)")),
			mcp.WithString(
				"repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString(
				"workflow_id",
				mcp.Required(),
				mcp.Description("Workflow file name (e.g., 'ci.yml') or numeric workflow ID"),
			),
			mcp.WithString(
				"ref",
				mcp.Required(),
				mcp.Description("Git branch or tag to run the workflow on (e.g., 'main', 'develop')"),
			),
			mcp.WithTitleAnnotation("Trigger Workflow"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),     // each trigger creates a new run
			mcp.WithOpenWorldHintAnnotation(true),
		),
		handleTriggerWorkflow,
	)

	// --- TOOL 6: List Workflow Runs ---
	mcpServer.AddTool(
		mcp.NewTool(
			"github_list_workflow_runs",
			mcp.WithDescription("List recent workflow runs (CI/CD pipeline executions) for a repository. "+
				"Shows status, conclusion, branch, and timing information."),
			mcp.WithString(
				"owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString(
				"repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithNumber(
				"per_page",
				mcp.Description("Number of runs to return (1-100, default 10)"),
			),
			mcp.WithTitleAnnotation("List Workflow Runs"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		handleListWorkflowRuns,
	)

	// --- TOOL 7: Get Repository Info ---
	mcpServer.AddTool(
		mcp.NewTool(
			"github_get_repo",
			mcp.WithDescription("Get detailed information about a specific GitHub repository."),
			mcp.WithString(
				"owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString(
				"repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithTitleAnnotation("Get Repository Details"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		handleGetRepo,
	)

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}


//Handler: List repositories
func handleListRepos(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	perPage := getNumberParam(request, "per_page", 30)
	page := getNumberParam(request, "page", 1)

	repos, _, err := ghClient.Repositories.ListByAuthenticatedUser(ctx,
		&github.RepositoryListByAuthenticatedUserOptions{
			Sort: "updated",
			ListOptions: github.ListOptions{
				PerPage: perPage,
				Page:    page,
			},
		},
	)
	// In Go, you ALWAYS check errors immediately after the call.
	// No try/catch - explicit error checking is the Go way.
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list repositories: %v", err)), nil
	}

	// strings.Builder is Go's efficient way to build strings incrementally.
	// Much better than str += "..." in a loop (strings are immutable in Go,
	// each += creates a new string in memory).
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d repositories:\n\n", len(repos)))
	for _, repo := range repos {
		// repo.GetName() is a safe getter. If repo were nil (null),
		// accessing repo.Name would PANIC (crash). GetXxx() returns
		// zero value ("", 0, false) if pointer is nil.
		visibility := "public"
		if repo.GetPrivate() {
			visibility = "private"
		}

		result.WriteString(fmt.Sprintf("- **%s** [%s]\n", repo.GetName(), visibility))

		description := repo.GetDescription()
		if description != "" {
			result.WriteString(fmt.Sprintf("  Description: %s\n", description))
		}

		result.WriteString(fmt.Sprintf("  URL: %s\n", repo.GetHTMLURL()))
		result.WriteString(fmt.Sprintf("  Language: %s | Stars: %d | Forks: %d\n\n",
			repo.GetLanguage(), repo.GetStargazersCount(), repo.GetForksCount()))
	}

	// mcp.NewToolResultText = successful response with text content.
	// Second return nil = "no error occurred".
	return mcp.NewToolResultText(result.String()), nil

}

// --- Handler: Create repoository ---
func handleCreateRepo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	name := getStringParam(request, "name")
	if name == "" {
		return mcp.NewToolResultError("Repository name is required"), nil
	}

	description := getStringParam(request, "description")
	private := getBoolParam(request, "private", false)
	autoInit := getBoolParam(request, "auto_init", false)

	// &github.Repository{...} creates a new Repository struct.
	//
	// github.Ptr() converts a plain value to a pointer.
	// go-github uses pointers for ALL fields because:
	// 1. nil = "don't set this field" vs &false = "set to false"
	// 2. nil fields are omitted from JSON, zero values are included
	// Similar to how GraphQL handles null vs undefined.
	repo, _, err := ghClient.Repositories.Create(ctx,
		"", // empty string = create for authenticated user (not an org)
		&github.Repository{
			Name:        github.Ptr(name),
			Description: github.Ptr(description),
			Private:     github.Ptr(private),
			AutoInit:    github.Ptr(autoInit),
		},
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create repository: %v", err)), nil
	}

	result := fmt.Sprintf("Repository created successfully!\n\n"+
		"- Name: %s\n"+
		"- Visibility: %s\n"+
		"- URL: %s\n"+
		"- Clone URL: %s\n",
		repo.GetFullName(),
		repo.GetVisibility(),
		repo.GetHTMLURL(),
		repo.GetCloneURL(),
	)

	return mcp.NewToolResultText(result), nil
}
// =============================================================================
// Handler: Delete Repository
// =============================================================================
func handleDeleteRepo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	owner := getStringParam(request, "owner")
	repo := getStringParam(request, "repo")

	if owner == "" || repo == "" {
		return mcp.NewToolResultError("Both 'owner' and 'repo' are required"), nil
	}

	// Delete returns only (Response, error) - no repo data back.
	// Makes sense since the repo no longer exists after deletion.
	_, err := ghClient.Repositories.Delete(ctx, owner, repo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to delete repository '%s/%s': %v", owner, repo, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Repository '%s/%s' has been permanently deleted.", owner, repo)), nil
}

func handleUpdateVisibility(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	owner := getStringParam(request, "owner")
	repo := getStringParam(request, "repo")
	visibility := getStringParam(request, "visibility")

	if owner == "" || repo == "" || visibility == "" {
		return mcp.NewToolResultError("'owner', 'repo', and 'visibility' are all required"), nil
	}

	// Validate visibility value
	visibility = strings.ToLower(visibility)
	if visibility != "public" && visibility != "private" {
		return mcp.NewToolResultError("Visibility must be 'public' or 'private'"), nil
	}

	// Repositories.Edit updates a repository's settings.
	// We only set the fields we want to change.
	updatedRepo, _, err := ghClient.Repositories.Edit(ctx, owner, repo,
		&github.Repository{
			Visibility: github.Ptr(visibility),
		},
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to update visibility for '%s/%s': %v", owner, repo, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Repository '%s' visibility changed to '%s'.\nURL: %s",
		updatedRepo.GetFullName(),
		updatedRepo.GetVisibility(),
		updatedRepo.GetHTMLURL(),
	)), nil
}

// =============================================================================
// Handler: Trigger GitHub Actions Workflow
// =============================================================================
func handleTriggerWorkflow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	owner := getStringParam(request, "owner")
	repo := getStringParam(request, "repo")
	workflowID := getStringParam(request, "workflow_id")
	ref := getStringParam(request, "ref")

	if owner == "" || repo == "" || workflowID == "" || ref == "" {
		return mcp.NewToolResultError(
			"'owner', 'repo', 'workflow_id', and 'ref' are all required"), nil
	}

	// ghClient.Actions is the "Actions service" for GitHub Actions endpoints.
	//
	// CreateWorkflowDispatchEventByFileName triggers a workflow by file name
	// (e.g., "ci.yml"). The workflow YAML must have:
	//   on:
	//     workflow_dispatch:
	_, err := ghClient.Actions.CreateWorkflowDispatchEventByFileName(ctx,
		owner, repo, workflowID,
		github.CreateWorkflowDispatchEventRequest{
			Ref: ref,
		},
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to trigger workflow '%s' on '%s/%s': %v",
			workflowID, owner, repo, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Workflow '%s' triggered successfully on branch '%s' for %s/%s.\n"+
			"Check status: https://github.com/%s/%s/actions",
		workflowID, ref, owner, repo, owner, repo)), nil
}

// =============================================================================
// Handler: List Workflow Runs
// =============================================================================
func handleListWorkflowRuns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	owner := getStringParam(request, "owner")
	repo := getStringParam(request, "repo")
	perPage := getNumberParam(request, "per_page", 10)

	if owner == "" || repo == "" {
		return mcp.NewToolResultError("'owner' and 'repo' are required"), nil
	}

	// ListWorkflowRunsByRepo returns all workflow runs for a repository.
	runs, _, err := ghClient.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo,
		&github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{
				PerPage: perPage,
			},
		},
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to list workflow runs for '%s/%s': %v", owner, repo, err)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Workflow runs for %s/%s (showing %d of %d total):\n\n",
		owner, repo, len(runs.WorkflowRuns), runs.GetTotalCount()))

	for _, run := range runs.WorkflowRuns {
		status := run.GetStatus()
		conclusion := run.GetConclusion()

		statusDisplay := status
		if conclusion != "" {
			statusDisplay = fmt.Sprintf("%s (%s)", status, conclusion)
		}

		result.WriteString(fmt.Sprintf("- **%s** [%s]\n", run.GetName(), statusDisplay))
		result.WriteString(fmt.Sprintf("  Branch: %s | Event: %s\n",
			run.GetHeadBranch(), run.GetEvent()))
		result.WriteString(fmt.Sprintf("  Started: %s\n",
			run.GetCreatedAt().Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("  URL: %s\n\n", run.GetHTMLURL()))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// =============================================================================
// Handler: Get Repository Details
// =============================================================================
func handleGetRepo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	owner := getStringParam(request, "owner")
	repo := getStringParam(request, "repo")

	if owner == "" || repo == "" {
		return mcp.NewToolResultError("'owner' and 'repo' are required"), nil
	}

	repoInfo, _, err := ghClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Failed to get repository '%s/%s': %v", owner, repo, err)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Repository: %s\n\n", repoInfo.GetFullName()))
	result.WriteString(fmt.Sprintf("- Description: %s\n", repoInfo.GetDescription()))
	result.WriteString(fmt.Sprintf("- Visibility: %s\n", repoInfo.GetVisibility()))
	result.WriteString(fmt.Sprintf("- Default Branch: %s\n", repoInfo.GetDefaultBranch()))
	result.WriteString(fmt.Sprintf("- Language: %s\n", repoInfo.GetLanguage()))
	result.WriteString(fmt.Sprintf("- Stars: %d\n", repoInfo.GetStargazersCount()))
	result.WriteString(fmt.Sprintf("- Forks: %d\n", repoInfo.GetForksCount()))
	result.WriteString(fmt.Sprintf("- Open Issues: %d\n", repoInfo.GetOpenIssuesCount()))
	result.WriteString(fmt.Sprintf("- Created: %s\n",
		repoInfo.GetCreatedAt().Format("2006-01-02")))
	result.WriteString(fmt.Sprintf("- Last Updated: %s\n",
		repoInfo.GetUpdatedAt().Format("2006-01-02")))
	result.WriteString(fmt.Sprintf("- URL: %s\n", repoInfo.GetHTMLURL()))
	result.WriteString(fmt.Sprintf("- Clone (HTTPS): %s\n", repoInfo.GetCloneURL()))
	result.WriteString(fmt.Sprintf("- Clone (SSH): %s\n", repoInfo.GetSSHURL()))

	result.WriteString("\nFeatures:\n")
	result.WriteString(fmt.Sprintf("- Issues: %v\n", repoInfo.GetHasIssues()))
	result.WriteString(fmt.Sprintf("- Wiki: %v\n", repoInfo.GetHasWiki()))
	result.WriteString(fmt.Sprintf("- Pages: %v\n", repoInfo.GetHasPages()))
	result.WriteString(fmt.Sprintf("- Archived: %v\n", repoInfo.GetArchived()))

	return mcp.NewToolResultText(result.String()), nil
}
// =============================================================================
// Helper Functions
// =============================================================================
// These extract typed values from the MCP request arguments map safely.

// getStringParam extracts a string parameter from the request.
// map access in Go returns (value, ok) where ok tells you if the key exists.
//
// Type assertion: val.(string) converts any -> string.
// Using comma-ok pattern (str, ok := val.(string)) is safe -
// ok is false instead of panicking if val isn't a string.
func getStringParam(request mcp.CallToolRequest, key string) string {
	args := request.GetArguments()

	// val = the value, ok = true if key was found
	val, ok := args[key]
	if !ok {
		return ""
	}

	// Type assertion with comma-ok pattern
	str, ok := val.(string)
	if !ok {
		return ""
	}

	return str
}

// getNumberParam extracts a numeric parameter as int.
// JSON numbers come through as float64 in Go (same as JavaScript!).
func getNumberParam(request mcp.CallToolRequest, key string, defaultVal int) int {
	args := request.GetArguments()

	val, ok := args[key]
	if !ok {
		return defaultVal
	}

	// JSON numbers are always float64 in Go's encoding/json.
	// int() converts: int(3.0) = 3
	num, ok := val.(float64)
	if !ok {
		return defaultVal
	}

	return int(num)
}

// getBoolParam extracts a boolean parameter with a default.
func getBoolParam(request mcp.CallToolRequest, key string, defaultVal bool) bool {
	args := request.GetArguments()

	val, ok := args[key]
	if !ok {
		return defaultVal
	}

	b, ok := val.(bool)
	if !ok {
		return defaultVal
	}

	return b
}