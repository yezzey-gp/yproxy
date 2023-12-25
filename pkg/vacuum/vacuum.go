package vacuum

// Greenplum + yezzey specific logic for external storage vacuuming
// Yezzey stores AO/AOCS relations data in one or several files in external
// storage (chunks). Chunk considered to be obsolete when
// table referencing this chunk was deleted and all backups (WAL-G) which
// contains this table was deleted

// TODO: implement
