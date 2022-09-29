package builder

// KanikoCacheDir is the cache directory for Kaniko builds (mounted into the Kaniko pod).
const KanikoCacheDir = "/kaniko/cache"
const KanikoPVCName = "KanikoPersistentVolumeClaim"
const KanikoBuildCacheEnabled = "KanikoBuildCacheEnabled"
const KanikoExecutorImage = "KanikoExecutorImage"
const KanikoWarmerImage = "KanikoWarmerImage"
const KanikoDefaultExecutorImageName = "gcr.io/kaniko-project/executor"
const KanikoDefaultWarmerImageName = "gcr.io/kaniko-project/warmer"

var kanikoSupportedOptions = map[string]PublishStrategyOption{
	KanikoPVCName: {
		Name:        KanikoPVCName,
		description: "The name of the PersistentVolumeClaim",
	},
	KanikoBuildCacheEnabled: {
		Name:         KanikoBuildCacheEnabled,
		description:  "To enable or disable the Kaniko cache",
		defaultValue: "false",
	},
	KanikoExecutorImage: {
		Name:         KanikoExecutorImage,
		description:  "The docker image of the Kaniko executor",
		defaultValue: KanikoDefaultExecutorImageName,
	},
	KanikoWarmerImage: {
		Name:         KanikoWarmerImage,
		description:  "The docker image of the Kaniko warmer",
		defaultValue: KanikoDefaultWarmerImageName,
	},
}
