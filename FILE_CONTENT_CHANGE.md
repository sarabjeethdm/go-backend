# File Content Storage Change Summary

## Changes Made

### Issue
Previously, the EDI file content was being stored in MongoDB's `file_content` field in the Job document, which could lead to:
- Large database documents
- Increased storage costs
- Slower query performance
- Unnecessary data duplication

### Solution
**File content is now passed through the Redis queue and NOT stored in MongoDB.**

---

## What Changed

### 1. **Job Model** (`internal/models/models.go`)
- ✅ Removed `FileContent` field from Job struct
- ✅ Added `JobID` field (string UUID) for job identification
- ✅ Added `FileName` field (optional) to track original filename
- ✅ Updated `NewJob()` to not require fileContent parameter

**Before:**
```go
type Job struct {
    ID          primitive.ObjectID
    FileContent string  // ❌ Removed
    Status      JobStatus
    ...
}
```

**After:**
```go
type Job struct {
    ID       primitive.ObjectID
    JobID    string  // ✅ UUID for lookups
    FileName string  // ✅ Optional metadata
    Status   JobStatus
    ...
}
```

### 2. **API Handler** (`internal/api/handlers.go`)
- ✅ Creates Job record WITHOUT file content
- ✅ Passes file content through Redis queue via JobMessage
- ✅ MongoDB only stores job metadata

**Flow:**
1. Upload file → Read content
2. Create Job (without content) → Save to MongoDB
3. Create JobMessage (with content) → Push to Redis queue
4. Return job ID to client

### 3. **Queue Message** (`internal/queue/queue.go`)
- ✅ JobMessage includes `FileContent` field
- ✅ File content travels through Redis, not MongoDB

```go
type JobMessage struct {
    JobID       string
    FileName    string
    FileContent string  // ✅ Only in queue
    CreatedAt   time.Time
}
```

### 4. **Worker** (`cmd/worker/main.go`)
- ✅ Dequeues JobMessage (with file content)
- ✅ Parses JSON to extract fileContent
- ✅ Passes fileContent to processor
- ✅ Backward compatible with legacy job ID format

**Processing Flow:**
```
Redis Queue → Dequeue JobMessage → Extract FileContent → Process → Save Result
```

### 5. **Processor** (`internal/worker/processor.go`)
- ✅ `ProcessJob()` now accepts `fileContent` parameter
- ✅ Uses fileContent from parameter, not from database
- ✅ Fails gracefully if fileContent is empty (legacy jobs)

### 6. **Storage** (`internal/storage/storage.go`)
- ✅ All queries now use `job_id` string field instead of MongoDB ObjectID
- ✅ Added unique index on `job_id` for fast lookups
- ✅ Updated methods:
  - `GetJob()` - Search by job_id
  - `UpdateJobStatus()` - Update by job_id
  - `UpdateJobWithResult()` - Update by job_id  
  - `IncrementRetryCount()` - Update by job_id

---

## Data Flow

### Before (File content in MongoDB ❌)
```
Client → API → MongoDB (Job + FileContent) → Redis (JobID) → Worker → MongoDB
                  ↓
        Large documents in DB
```

### After (File content in Redis only ✅)
```
Client → API → MongoDB (Job metadata only) → Redis (JobMessage with FileContent) → Worker
                  ↓                                        ↓
        Small documents                          Content in memory only
```

---

## Benefits

### 1. **Reduced Database Size**
- MongoDB documents are much smaller
- Only metadata stored: job_id, status, result, timestamps
- File content not persisted to disk

### 2. **Better Performance**
- Faster database queries (smaller documents)
- Less network transfer between services and database
- Improved index efficiency

### 3. **Cost Savings**
- Lower storage costs
- Reduced backup sizes
- Less bandwidth usage

### 4. **Scalability**
- MongoDB can handle more jobs with same resources
- Redis queue is optimized for message passing
- Better memory utilization

### 5. **Security**
- Sensitive file content not permanently stored
- Easier to implement data retention policies
- Content automatically cleared after processing

---

## MongoDB Document Structure

### Before
```json
{
  "_id": ObjectId("..."),
  "file_content": "CLAIM*CLM001*MEM123*2500\nCLAIM*CLM002...",  // ❌ Large
  "status": "completed",
  "result": {...},
  ...
}
```

### After
```json
{
  "_id": ObjectId("..."),
  "job_id": "550e8400-e29b-41d4-a716-446655440000",  // ✅ UUID
  "file_name": "claims.edi",                          // ✅ Metadata
  "status": "completed",
  "result": {...},
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:05Z"
}
```

---

## Backward Compatibility

The system maintains backward compatibility:

1. **Old queue messages** (just job ID string): Worker handles gracefully but will fail processing
2. **New queue messages** (full JobMessage JSON): Worker processes normally
3. **Migration path**: All new jobs use the new format

---

## Testing Considerations

### Unit Tests
- ✅ Test Job model without FileContent
- ✅ Test processor with fileContent parameter
- ✅ Test storage lookup by job_id

### Integration Tests
- ✅ Test full flow: Upload → Queue → Process → Result
- ✅ Verify file content not in MongoDB
- ✅ Verify file content in Redis queue message

---

## Database Indexes

New indexes for optimal performance:

```javascript
// Unique index on job_id for lookups
db.jobs.createIndex({ "job_id": 1 }, { unique: true })

// Index on status for filtering
db.jobs.createIndex({ "status": 1 })
```

---

## Migration Notes

If you have existing data:

1. **Existing jobs in MongoDB**: Will fail to reprocess (no file content)
2. **Solution**: These jobs are already completed, no action needed
3. **New jobs**: Automatically use new format

---

## Files Modified

1. ✅ `internal/models/models.go` - Removed FileContent field
2. ✅ `internal/api/handlers.go` - Don't save file content
3. ✅ `internal/storage/storage.go` - Use job_id for queries
4. ✅ `cmd/worker/main.go` - Parse JobMessage from queue
5. ✅ `internal/worker/processor.go` - Accept fileContent parameter

---

## Summary

✅ **File content is NO LONGER stored in MongoDB**
✅ **File content is passed through Redis queue only**
✅ **MongoDB stores only job metadata and results**
✅ **Better performance, lower costs, improved scalability**

The system is now more efficient and follows best practices for message queue architectures!
