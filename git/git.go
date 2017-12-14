package git

import (
	"github.com/boltdb/bolt"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/blobs"
)

type Repo struct {
	ds  *ssb.DataStore
	Ref ssb.Ref
}

type RepoRoot struct {
	ssb.MessageBody
	Name string `json:"name"`
}

/*
{
  type: 'git-update',
  repo: MsgId,
  repoBranch: [ MsgId ]?,
  refsBranch: [ MsgId ]?,
  refs: { <ref>: String|null }?,
  packs: [ BlobLink ]?,
  indexes: [ BlobLink ]?,
  head: string?,
  commits: [ {
    sha1: String,
    title: String,
    body: String?,
    parents: [ String ]?,
  } ]?,
  commits_more: Number?,
  num_objects: Number?,
  object_ids: [ String ]?,
}
*/
type Commit struct {
	Sha1    string   `json:"sha1"`
	Title   string   `json:"title"`
	Body    string   `json:"body,omitempty"`
	Parents []string `json:"parents,omitempty"`
}

type RepoUpdate struct {
	ssb.MessageBody
	Repo        ssb.Ref            `json:"repo"`
	RepoBranch  []ssb.Ref          `json:"repoBranch,omitempty"`
	RefsBranch  []ssb.Ref          `json:"refsBranch,omitempty"`
	Refs        map[ssb.Ref]string `json:"refs,omitempty"`
	Packs       []blobs.BlobLink   `json:"packs,omitempty"`
	Indexes     []blobs.BlobLink   `json:"indexes,omitempty"`
	Head        string             `json:"Head,omitempty"`
	Commits     []Commit           `json:"commits,omitempty"`
	CommitsMore int                `json:"commits_more,omitempty"`
	NumObjects  int                `json:"num_objects,omitempty"`
	ObjectIDs   []string           `json:"object_ids,omitempty"`
}

type RepoIssue struct {
	ssb.MessageBody
	Project ssb.Ref `json:"project"`
	Text    string  `json:"text"`
}

func init() {
	ssb.RebuildClearHooks["git"] = func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte("repos"))
		return nil
	}
	ssb.AddMessageHooks["git"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if _, ok := mb.(*RepoRoot); ok {
			ReposBucket, err := tx.CreateBucketIfNotExists([]byte("repos"))
			if err != nil {
				return err
			}
			repoBucket, err := ReposBucket.CreateBucketIfNotExists(m.Key().DBKey())
			if err != nil {
				return err
			}
			err = repoBucket.Put([]byte("info"), m.Compress())
			if err != nil {
				return err
			}
			return nil
		}
		if update, ok := mb.(*RepoUpdate); ok {
			ReposBucket, err := tx.CreateBucketIfNotExists([]byte("repos"))
			if err != nil {
				return err
			}
			repoBucket, err := ReposBucket.CreateBucketIfNotExists(update.Repo.DBKey())
			if err != nil {
				return err
			}
			updateBucket, err := repoBucket.CreateBucketIfNotExists([]byte("updates"))
			if err != nil {
				return err
			}
			err = updateBucket.Put(m.Key().DBKey(), []byte{})
			if err != nil {
				return err
			}
			blobBucket, err := repoBucket.CreateBucketIfNotExists([]byte("blobs"))
			if err != nil {
				return err
			}
			for _, pack := range update.Packs {
				err = blobBucket.Put(pack.Link.DBKey(), []byte{})
				if err != nil {
					return err
				}
			}
			for _, index := range update.Indexes {
				err = blobBucket.Put(index.Link.DBKey(), []byte{})
				if err != nil {
					return err
				}
			}
		}
		if issue, ok := mb.(*RepoIssue); ok {
			ReposBucket, err := tx.CreateBucketIfNotExists([]byte("repos"))
			if err != nil {
				return err
			}
			repoBucket, err := ReposBucket.CreateBucketIfNotExists(issue.Project.DBKey())
			if err != nil {
				return err
			}
			issueBucket, err := repoBucket.CreateBucketIfNotExists([]byte("issues"))
			if err != nil {
				return err
			}
			err = issueBucket.Put(m.Key().DBKey(), []byte{})
			if err != nil {
				return err
			}
		}

		return nil
	}
	ssb.MessageTypes["git-repo"] = func(mb ssb.MessageBody) interface{} {
		return &RepoRoot{MessageBody: mb}
	}
	ssb.MessageTypes["git-update"] = func(mb ssb.MessageBody) interface{} {
		return &RepoUpdate{MessageBody: mb}
	}
	ssb.MessageTypes["issue"] = func(mb ssb.MessageBody) interface{} {
		return &RepoIssue{MessageBody: mb}
	}
}

func Get(ds *ssb.DataStore, r ssb.Ref) *Repo {
	msg := ds.Get(nil, r)
	if msg == nil || msg.Type() != "git-repo" {
		return nil
	}
	return &Repo{
		ds:  ds,
		Ref: r,
	}
}

func (repo *Repo) WantAll() {
	repo.ds.DB().View(func(tx *bolt.Tx) error {

		ReposBucket := tx.Bucket([]byte("repos"))
		if ReposBucket == nil {
			return nil
		}
		repoBucket := ReposBucket.Bucket(repo.Ref.DBKey())
		if repoBucket == nil {
			return nil
		}
		blobBucket := repoBucket.Bucket([]byte("blobs"))
		if blobBucket == nil {
			return nil
		}
		blobBucket.ForEach(func(k, v []byte) error {
			r := ssb.DBRef(k)
			blobs.Get(repo.ds).Want(r)

			return nil
		})
		return nil
	})
}

func (repo *Repo) ListBlobs() (b []ssb.Ref) {
	repo.ds.DB().View(func(tx *bolt.Tx) error {

		ReposBucket := tx.Bucket([]byte("repos"))
		if ReposBucket == nil {
			return nil
		}
		repoBucket := ReposBucket.Bucket(repo.Ref.DBKey())
		if repoBucket == nil {
			return nil
		}
		blobBucket := repoBucket.Bucket([]byte("blobs"))
		if blobBucket == nil {
			return nil
		}
		blobBucket.ForEach(func(k, v []byte) error {
			r := ssb.DBRef(k)
			b = append(b, r)
			return nil
		})
		return nil
	})
	return
}

func (repo *Repo) ListUpdates() (b []ssb.Ref) {
	repo.ds.DB().View(func(tx *bolt.Tx) error {

		ReposBucket := tx.Bucket([]byte("repos"))
		if ReposBucket == nil {
			return nil
		}
		repoBucket := ReposBucket.Bucket(repo.Ref.DBKey())
		if repoBucket == nil {
			return nil
		}
		blobBucket := repoBucket.Bucket([]byte("updates"))
		if blobBucket == nil {
			return nil
		}
		blobBucket.ForEach(func(k, v []byte) error {
			r := ssb.DBRef(k)
			b = append(b, r)
			return nil
		})
		return nil
	})
	return
}

func (repo *Repo) Issues() (issues []*ssb.SignedMessage) {
	repo.ds.DB().View(func(tx *bolt.Tx) error {

		ReposBucket := tx.Bucket([]byte("repos"))
		if ReposBucket == nil {
			return nil
		}
		repoBucket := ReposBucket.Bucket(repo.Ref.DBKey())
		if repoBucket == nil {
			return nil
		}
		blobBucket := repoBucket.Bucket([]byte("issues"))
		if blobBucket == nil {
			return nil
		}
		blobBucket.ForEach(func(k, v []byte) error {
			r := ssb.DBRef(k)
			issues = append(issues, repo.ds.Get(tx, r))
			return nil
		})
		return nil
	})
	return
}
