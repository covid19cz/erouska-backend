package constants

//TopicNameImportKeys Topic for enqueuing keys to be imported.
const TopicNameImportKeys = "efgs-import-keys"

//TopicNameContinueYesterdayDownloading Topic for postponing download of yesterdays keys.
const TopicNameContinueYesterdayDownloading = "efgs-postponed-yesterdays-downloading"

//MutexNameDownloadAndSaveKeys Name for mutex for EFGS keys downloading.
const MutexNameDownloadAndSaveKeys = "download-and-save-keys"

//RedisKeyNextBatch Key for next download batch metadata.
const RedisKeyNextBatch = "nextDownloadBatch"
