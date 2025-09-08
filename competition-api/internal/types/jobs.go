package types

type ArchivedFile string

const (
	FileFuzzOutHead           ArchivedFile = "fuzz.out"
	FileFuzzOutBase           ArchivedFile = "fuzz.out_base_repo"
	FilePOVTrigger            ArchivedFile = "pov_trigger"
	FilePatch                 ArchivedFile = "patch"
	FileUnstrippedHeadTarball ArchivedFile = "unstripped_head_tarball"
	FileUnstrippedBaseTarball ArchivedFile = "unstripped_base_tarball"
	FileDiffTarball           ArchivedFile = "diff_tarball"
	FileStrippedRepoTarball   ArchivedFile = "stripped_repo_tarball"
	FileUnstrippedRepoTarball ArchivedFile = "unstripped_repo_tarball"
	FileOSSFuzzTarball        ArchivedFile = "oss_fuzz_tarball"
	FileSARIFSubmission       ArchivedFile = "sarif_submission"
	FileSARIFBroadcast        ArchivedFile = "sarif_broadcast"
	FileFreeformPOV           ArchivedFile = "freeform_pov"
)

type (
	ResultContext string

	Blob struct {
		PresignedURL *string `json:"presigned_url"`
		ObjectName   string  `json:"object_name"`
	}

	JobArtifact struct {
		Blob         Blob          `json:"blob"`
		ArchivedFile ArchivedFile  `json:"archived_file"`
		Filename     string        `json:"filename"`
		Context      ResultContext `json:"context"`
	}

	JobResult struct {
		StdoutBlob Blob          `json:"stdout_blob,omitempty"`
		StderrBlob Blob          `json:"stderr_blob,omitempty"`
		ExitCode   *int          `json:"return_code,omitempty"`
		Context    ResultContext `json:"context,omitempty"`
		Cmd        []string      `json:"cmd,omitempty"`
	}

	JobResponse struct {
		JobID                     string           `json:"job_id"                      validate:"required"`
		Status                    SubmissionStatus `json:"status"                      validate:"required"`
		FunctionalityTestsPassing *bool            `json:"functionality_tests_passing"`
		Results                   []JobResult      `json:"results"`
		Artifacts                 []JobArtifact    `json:"artifacts"`
	}

	JobArgs struct {
		TaskID *string `json:"task_id" validate:"omitempty,uuid_rfc4122" format:"uuid"`

		Architecture *string `json:"architecture" validate:"required"`

		FuzzerName  *string `json:"fuzzer_name"   validate:"required_with=TestcaseB64 TestcaseHash"`
		Sanitizer   *string `json:"sanitizer"     validate:"required_with=TestcaseB64 TestcaseHash"`
		Engine      *string `json:"engine"        validate:"required_with=TestcaseB64 TestcaseHash"`
		TestcaseB64 *string `json:"testcase_b64"                                                    format:"base64"`
		// Lower priority than b64
		TestcaseHash *string `json:"testcase_hash"`

		PatchB64 *string `json:"patch_b64"        format:"base64"`
		// Lower priority than b64
		PatchHash      *string `json:"patch_hash"`
		SkipPatchTests *bool   `json:"skip_patch_tests"`

		HeadRepoTarballURL *string `json:"repo_tarball_url"     validate:"required_without=TaskID"`
		OssFuzzTarballURL  *string `json:"oss_fuzz_tarball_url" validate:"required_without=TaskID"`
		Focus              *string `json:"focus"                validate:"required_without=TaskID"`
		ProjectName        *string `json:"project_name"         validate:"required_without=TaskID"`
		BaseTarballURL     *string `json:"diff_tarball_url"`
		MemoryGB           *int    `json:"memory_gb"            validate:"required_without=TaskID"`
		CPUs               *int    `json:"cpus"                 validate:"required_without=TaskID"`

		CacheKey      *string `json:"cache_key"      validate:"required"`
		OverrideCache *bool   `json:"override_cache" validate:"required"`
	}
)

const (
	ResultCtxHeadRepoTest ResultContext = "pov_test_head_repo"
	ResultCtxBaseRepoTest ResultContext = "delta_test_base_repo"
)
