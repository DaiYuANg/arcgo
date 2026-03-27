package authx_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/brianvoe/gofakeit/v7"
)

type benchmarkDatasetCredential struct {
	UserID string
}

type benchmarkDatasetQuery struct {
	userID   string
	action   string
	resource string
	allowed  bool
}

type benchmarkDataset struct {
	userPermissions map[string]map[string]struct{}
	queries         []benchmarkDatasetQuery
}

var benchmarkDatasetCases = []benchmarkCase{
	{name: "NoHook", withHook: false},
	{name: "WithHook", withHook: true},
}

func BenchmarkEngineCheckThenCan10kUsers10kPermissions(b *testing.B) {
	ctx := context.Background()
	dataset := newBenchmarkDataset(10_000, 10_000, 16, 4_096)

	for _, benchCase := range benchmarkDatasetCases {
		b.Run(benchCase.name, func(b *testing.B) {
			benchmarkDatasetCheckThenCan(ctx, b, dataset, benchCase.withHook)
		})
	}
}

func BenchmarkEngineCheckThenCan10kUsers10kPermissionsParallel(b *testing.B) {
	ctx := context.Background()
	dataset := newBenchmarkDataset(10_000, 10_000, 16, 4_096)

	for _, benchCase := range benchmarkDatasetCases {
		b.Run(benchCase.name, func(b *testing.B) {
			benchmarkDatasetCheckThenCanParallel(ctx, b, dataset, benchCase.withHook)
		})
	}
}

func newBenchmarkDatasetEngine(dataset benchmarkDataset, withHook bool) *authx.Engine {
	manager := authx.NewProviderManager(
		authx.NewAuthenticationProviderFunc(func(
			_ context.Context,
			credential benchmarkDatasetCredential,
		) (authx.AuthenticationResult, error) {
			if _, ok := dataset.userPermissions[credential.UserID]; !ok {
				return authx.AuthenticationResult{}, authx.ErrUnauthenticated
			}
			return authx.AuthenticationResult{
				Principal: authx.Principal{ID: credential.UserID},
			}, nil
		}),
	)

	authorizer := authx.AuthorizerFunc(func(_ context.Context, input authx.AuthorizationModel) (authx.Decision, error) {
		principal, ok := input.Principal.(authx.Principal)
		if !ok || principal.ID == "" {
			return authx.Decision{Allowed: false, Reason: "invalid_principal"}, nil
		}

		userPermissions, ok := dataset.userPermissions[principal.ID]
		if !ok {
			return authx.Decision{Allowed: false, Reason: "user_not_found"}, nil
		}

		_, allowed := userPermissions[permissionKey(input.Action, input.Resource)]
		if !allowed {
			return authx.Decision{Allowed: false, Reason: "no_permission"}, nil
		}
		return authx.Decision{Allowed: true}, nil
	})

	opts := []authx.EngineOption{
		authx.WithAuthenticationManager(manager),
		authx.WithAuthorizer(authorizer),
	}
	if withHook {
		opts = append(opts, authx.WithHook(noopHook{}))
	}
	return authx.NewEngine(opts...)
}

func newBenchmarkDataset(
	userCount int,
	permissionCount int,
	permissionsPerUser int,
	queryCount int,
) benchmarkDataset {
	randSource := gofakeit.New(42)
	permissions := buildBenchmarkPermissions(randSource, permissionCount)
	userIDs, userPermissions := buildBenchmarkUsers(randSource, userCount, permissionsPerUser, permissions)
	queries := buildBenchmarkQueries(randSource, userIDs, userPermissions, permissions, queryCount)

	return benchmarkDataset{
		userPermissions: userPermissions,
		queries:         queries,
	}
}

func samplePermission(randSource *gofakeit.Faker, assigned map[string]struct{}) string {
	target := randSource.Number(0, len(assigned)-1)
	for permission := range assigned {
		if target == 0 {
			return permission
		}
		target--
	}
	return ""
}

func permissionKey(action, resource string) string {
	return action + "|" + resource
}

func parsePermissionKey(key string) (string, string) {
	action, resource, found := strings.Cut(key, "|")
	if !found {
		return key, ""
	}
	return action, resource
}

func normalizeFakeToken(raw string) string {
	token := strings.ToLower(strings.TrimSpace(raw))
	token = strings.ReplaceAll(token, " ", "_")
	token = strings.ReplaceAll(token, "-", "_")
	if token == "" {
		return "x"
	}
	return token
}

func benchmarkDatasetCheckThenCan(
	ctx context.Context,
	b *testing.B,
	dataset benchmarkDataset,
	withHook bool,
) {
	b.Helper()

	engine := newBenchmarkDatasetEngine(dataset, withHook)
	queries := dataset.queries

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		runBenchmarkDatasetQuery(ctx, b, engine, queries[i%len(queries)])
	}
}

func benchmarkDatasetCheckThenCanParallel(
	ctx context.Context,
	b *testing.B,
	dataset benchmarkDataset,
	withHook bool,
) {
	b.Helper()

	engine := newBenchmarkDatasetEngine(dataset, withHook)
	queries := dataset.queries

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		queryIndex := 0
		for pb.Next() {
			runBenchmarkDatasetQuery(ctx, b, engine, queries[queryIndex%len(queries)])
			queryIndex++
		}
	})
}

func runBenchmarkDatasetQuery(
	ctx context.Context,
	b *testing.B,
	engine *authx.Engine,
	query benchmarkDatasetQuery,
) {
	b.Helper()

	result, err := engine.Check(ctx, benchmarkDatasetCredential{UserID: query.userID})
	if err != nil {
		b.Fatalf("check failed: %v", err)
	}

	decision, err := engine.Can(ctx, authx.AuthorizationModel{
		Principal: result.Principal,
		Action:    query.action,
		Resource:  query.resource,
	})
	if err != nil {
		b.Fatalf("can failed: %v", err)
	}
	if decision.Allowed != query.allowed {
		b.Fatalf("decision mismatch: allowed=%v expected=%v", decision.Allowed, query.allowed)
	}
}

func buildBenchmarkPermissions(randSource *gofakeit.Faker, permissionCount int) []string {
	permissions := make([]string, permissionCount)
	for i := range permissionCount {
		action := fmt.Sprintf("%s-%03d", normalizeFakeToken(randSource.Verb()), i/100)
		resource := fmt.Sprintf("%s-%03d", normalizeFakeToken(randSource.Noun()), i%100)
		permissions[i] = permissionKey(action, resource)
	}
	return permissions
}

func buildBenchmarkUsers(
	randSource *gofakeit.Faker,
	userCount int,
	permissionsPerUser int,
	permissions []string,
) ([]string, map[string]map[string]struct{}) {
	userIDs := make([]string, userCount)
	userPermissions := make(map[string]map[string]struct{}, userCount)
	for i := range userCount {
		userID := fmt.Sprintf("%s-%05d", normalizeFakeToken(randSource.Username()), i)
		userIDs[i] = userID
		userPermissions[userID] = pickAssignedPermissions(randSource, permissionsPerUser, permissions)
	}
	return userIDs, userPermissions
}

func pickAssignedPermissions(
	randSource *gofakeit.Faker,
	permissionsPerUser int,
	permissions []string,
) map[string]struct{} {
	assigned := make(map[string]struct{}, permissionsPerUser)
	for len(assigned) < permissionsPerUser {
		assigned[permissions[randSource.Number(0, len(permissions)-1)]] = struct{}{}
	}
	return assigned
}

func buildBenchmarkQueries(
	randSource *gofakeit.Faker,
	userIDs []string,
	userPermissions map[string]map[string]struct{},
	permissions []string,
	queryCount int,
) []benchmarkDatasetQuery {
	queries := make([]benchmarkDatasetQuery, queryCount)
	for i := range queryCount {
		queries[i] = buildBenchmarkQuery(randSource, userIDs, userPermissions, permissions, i)
	}
	return queries
}

func buildBenchmarkQuery(
	randSource *gofakeit.Faker,
	userIDs []string,
	userPermissions map[string]map[string]struct{},
	permissions []string,
	index int,
) benchmarkDatasetQuery {
	userID := userIDs[randSource.Number(0, len(userIDs)-1)]
	assigned := userPermissions[userID]
	permission, allowed := selectBenchmarkPermission(randSource, assigned, permissions, index)
	action, resource := parsePermissionKey(permission)

	return benchmarkDatasetQuery{
		userID:   userID,
		action:   action,
		resource: resource,
		allowed:  allowed,
	}
}

func selectBenchmarkPermission(
	randSource *gofakeit.Faker,
	assigned map[string]struct{},
	permissions []string,
	index int,
) (string, bool) {
	permission := samplePermission(randSource, assigned)
	if index%2 == 0 {
		return permission, true
	}

	for {
		candidate := permissions[randSource.Number(0, len(permissions)-1)]
		if _, exists := assigned[candidate]; !exists {
			return candidate, false
		}
	}
}
