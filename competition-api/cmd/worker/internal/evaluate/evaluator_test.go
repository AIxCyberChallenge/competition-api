package evaluate_test

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine"
	mockengine "github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/evaluate"
	mockextract "github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/extract/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/workerqueue"
	mockfetch "github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	mockqueue "github.com/aixcyberchallenge/competition-api/competition-api/internal/queue/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var fuzzTooling = "fuzztooling"
var headRepo = "headrepo"
var baseRepo = "baserepo"
var trigger = "trigger"
var patch = "patch"
var projectName = "projectname"
var focus = "focus"
var sanitizer = "sanitizer"
var harness = "harness"
var architecture = "architecture"
var engineName = "engine"
var entityType = types.JobTypeJob
var entityID = "entityID"
var commonEngineParams = engine.NewParams(
	sanitizer,
	architecture,
	engineName,
	harness,
	projectName,
	focus,
	identifier.LanguageSlice{identifier.LanguageC, identifier.LanguageJava},
)

func fetch(tempDir string, fetcher *mockfetch.MockFetcher, name string) (*gomock.Call, *os.File) {
	f, err := os.CreateTemp(tempDir, "file-*")
	fetch := fetcher.
		EXPECT().Fetch(gomock.Any(), name).
		Return(f, err).Times(1)
	return fetch, f
}

func fetchAndExtract(
	tempDir string,
	fetcher *mockfetch.MockFetcher,
	extractor *mockextract.MockExtractor,
	url string,
) (*gomock.Call, *string) {
	fetch, f := fetch(tempDir, fetcher, url)
	outdir := new(string)
	return extractor.
		EXPECT().
		Extract(gomock.Any(), gomock.Eq(f), gomock.Any()).
		Do(func(_ any, _ any, outDir string) {
			*outdir = outDir
		}).
		Times(1).
		After(fetch), outdir
}

func challengeDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(*engine.Params) {
	return func(actual *engine.Params) {
		expected := base.WithFuzzToolingDir(*fuzzToolingDir).WithRepo(resultContext, *repoDir)
		assert.Equal(t, &expected, actual, "wrong challenge data")
	}
}

func buildDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(context.Context, *engine.Params) {
	tester := challengeDataTest(t, base, fuzzToolingDir, repoDir, resultContext)
	return func(_ context.Context, actual *engine.Params) {
		tester(actual)
	}
}

func runDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(context.Context, *engine.Params, string, bool) {
	tester := challengeDataTest(t, base, fuzzToolingDir, repoDir, resultContext)
	return func(_ context.Context, actual *engine.Params, _ string, _ bool) {
		tester(actual)
	}
}

func patchDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(context.Context, *engine.Params, string) {
	tester := challengeDataTest(t, base, fuzzToolingDir, repoDir, resultContext)
	return func(_ context.Context, actual *engine.Params, _ string) {
		tester(actual)
	}
}

func checkDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(context.Context, *engine.Params) {
	tester := challengeDataTest(t, base, fuzzToolingDir, repoDir, resultContext)
	return func(_ context.Context, actual *engine.Params) {
		tester(actual)
	}
}

func testDataTest(
	t *testing.T,
	base *engine.Params,
	fuzzToolingDir *string,
	repoDir *string,
	resultContext types.ResultContext,
) func(context.Context, *engine.Params, bool) {
	tester := challengeDataTest(t, base, fuzzToolingDir, repoDir, resultContext)
	return func(_ context.Context, actual *engine.Params, _ bool) {
		tester(actual)
	}
}

// All args
func TestEvaluatorDeltaJobPatchEval(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).Times(0)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchTrigger, _ := fetch(tempDir, fetcher, trigger)
	fetchPatch, _ := fetch(tempDir, fetcher, patch)

	applyPatch := engineMock.
		EXPECT().
		ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(patchDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchPatch).After(fetchHeadRepo)
	buildPatch := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(applyPatch).After(checkData)
	_ = engineMock.
		EXPECT().
		RunPov(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(false)).
		Do(runDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildPatch).After(fetchTrigger).After(checkData)
	_ = engineMock.
		EXPECT().
		RunTests(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(testDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildPatch).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		baseRepo,
		trigger,
		patch,
		false,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// No Patch
func TestEvaluatorDeltaScanPov(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)
	fetchBaseRepo, baseRepoDir := fetchAndExtract(tempDir, fetcher, extractor, baseRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchTrigger, _ := fetch(tempDir, fetcher, trigger)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Any()).Times(0)
	buildBase := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, baseRepoDir, types.ResultCtxBaseRepoTest)).
		Times(1).
		After(fetchFuzzTooling).After(fetchBaseRepo).After(checkData)
	runBase := engineMock.
		EXPECT().
		RunPov(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(false)).
		Do(runDataTest(t, &commonEngineParams, fuzzToolingDir, baseRepoDir, types.ResultCtxBaseRepoTest)).
		Times(1).
		After(buildBase).After(fetchTrigger).After(checkData)

	buildHead := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchFuzzTooling).After(fetchHeadRepo).After(runBase).After(checkData)
	_ = engineMock.
		EXPECT().
		RunPov(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(runDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildHead).After(fetchTrigger).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		baseRepo,
		trigger,
		"",
		true,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// No Patch
// No Base
func TestEvaluatorFullScanPov(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchTrigger, _ := fetch(tempDir, fetcher, trigger)

	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).MaxTimes(0)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(patch)).MaxTimes(0)

	buildHead := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchFuzzTooling).After(fetchHeadRepo).After(checkData)
	_ = engineMock.
		EXPECT().
		RunPov(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(runDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildHead).After(fetchTrigger).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		"",
		trigger,
		"",
		true,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// No Trigger
func TestEvaluatorDeltaPatch(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchPatch, _ := fetch(tempDir, fetcher, patch)

	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).MaxTimes(0)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(trigger)).MaxTimes(0)

	applyPatch := engineMock.
		EXPECT().
		ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(patchDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchPatch).After(fetchHeadRepo)
	buildPatch := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(applyPatch).After(fetchFuzzTooling).After(checkData)
	_ = engineMock.
		EXPECT().
		RunTests(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(testDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildPatch).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		baseRepo,
		"",
		patch,
		false,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// No Trigger
// No Base
func TestEvaluatorFullPatch(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchPatch, _ := fetch(tempDir, fetcher, patch)

	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).MaxTimes(0)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(trigger)).MaxTimes(0)

	applyPatch := engineMock.
		EXPECT().
		ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(patchDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchPatch).After(fetchHeadRepo)
	buildPatch := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(applyPatch).After(fetchFuzzTooling).After(checkData)
	_ = engineMock.
		EXPECT().
		RunTests(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(testDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildPatch).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		"",
		"",
		patch,
		false,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEvaluatorFullPatchLowTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)

	expected := types.NewWorkerMsgFinal(
		entityType,
		entityID,
		types.SubmissionStatusInconclusive,
		nil,
	)
	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Eq(expected)).Times(1)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).AnyTimes()
	fetcher.EXPECT().
		Fetch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (io.ReadCloser, error) {
			return os.CreateTemp("", "file-*")
		}).
		AnyTimes()
	extractor.EXPECT().Extract(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engineMock.EXPECT().Build(gomock.Any(), gomock.Any()).AnyTimes()
	engineMock.EXPECT().ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	engineMock.EXPECT().Check(gomock.Any(), gomock.Any()).AnyTimes()
	engineMock.EXPECT().RunTests(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		"",
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*0)
	defer cancel()
	err := evaluator.Evaluate(
		timeoutCtx,
		fuzzTooling,
		headRepo,
		"",
		"",
		patch,
		false,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// No Trigger
// No Base
func TestEvaluatorFullPatchBuildFailed(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().
		Enqueue(gomock.Any(), gomock.Eq(types.NewWorkerMsgFinal(entityType, entityID, types.SubmissionStatusFailed, nil))).
		Times(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchPatch, _ := fetch(tempDir, fetcher, patch)

	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).MaxTimes(0)
	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(trigger)).MaxTimes(0)

	applyPatch := engineMock.
		EXPECT().
		ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(patchDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchPatch).After(fetchHeadRepo)
	_ = engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Return(workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, engine.ErrBuildingFailed)).
		Times(1).
		After(applyPatch).After(fetchFuzzTooling).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		"",
		"",
		patch,
		false,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// no base
// skip patch tests
func TestEvaluatorFullPatchSkipTests(t *testing.T) {
	tempDir := t.TempDir()
	ctrl := gomock.NewController(t)

	fetcher := mockfetch.NewMockFetcher(ctrl)
	extractor := mockextract.NewMockExtractor(ctrl)
	engineMock := mockengine.NewMockEngine(ctrl)
	queuer := mockqueue.NewMockQueuer(ctrl)
	queuer.EXPECT().Enqueue(gomock.Any(), gomock.Any()).MinTimes(1)

	fetchFuzzTooling, fuzzToolingDir := fetchAndExtract(tempDir, fetcher, extractor, fuzzTooling)
	fetchHeadRepo, headRepoDir := fetchAndExtract(tempDir, fetcher, extractor, headRepo)

	checkData := engineMock.
		EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Do(checkDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).After(fetchFuzzTooling).After(fetchHeadRepo)

	fetchPatch, _ := fetch(tempDir, fetcher, patch)
	fetchTrigger, _ := fetch(tempDir, fetcher, trigger)

	fetcher.EXPECT().Fetch(gomock.Any(), gomock.Eq(baseRepo)).MaxTimes(0)

	applyPatch := engineMock.
		EXPECT().
		ApplyPatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(patchDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(fetchPatch).After(fetchHeadRepo)
	buildPatch := engineMock.
		EXPECT().
		Build(gomock.Any(), gomock.Any()).
		Do(buildDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(applyPatch).After(fetchFuzzTooling).After(checkData)
	_ = engineMock.
		EXPECT().
		RunTests(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Do(testDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(0).
		After(buildPatch).After(checkData)
	_ = engineMock.
		EXPECT().
		RunPov(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(false)).
		Do(runDataTest(t, &commonEngineParams, fuzzToolingDir, headRepoDir, types.ResultCtxHeadRepoTest)).
		Times(1).
		After(buildPatch).After(fetchTrigger).After(checkData)

	evaluator := evaluate.NewEvaluator(
		fetcher,
		extractor,
		tempDir,
		engineMock,
		workerqueue.NewWorkerQueue(entityID, entityType, queuer),
	)

	err := evaluator.Evaluate(
		context.Background(),
		fuzzTooling,
		headRepo,
		"",
		trigger,
		patch,
		true,
		&commonEngineParams,
	)
	if err != nil {
		t.Fatal(err)
	}
}
