package git

import (
	"encoding/json"

	"github.com/boltdb/bolt"

	"github.com/andyleap/go-ssb"
)

type Repo struct {
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
	Packs       []ssb.Ref          `json:"packs,omitempty"`
	Indexes     []ssb.Ref          `json:"indexes,omitempty"`
	Head        string             `json:"Head,omitempty"`
	Commits     []Commit           `json:"commits,omitempty"`
	CommitsMore int                `json:"commits_more,omitempty"`
	NumObjects  int                `json:"num_objects,omitempty"`
	ObjectIDs   []string           `json:"object_ids,omitempty"`
}

func init() {
	ssb.RebuildClearHooks["git"] = func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte("repos"))
		return nil
	}
	ssb.AddMessageHooks["git"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mbr, ok := mb.(*Repo); ok {
			PubBucket, err := tx.CreateBucketIfNotExists([]byte("repos"))
			if err != nil {
				return err
			}
			buf, _ := json.Marshal(mbr)
			PubBucket.Put(m.Key().DBKey(), buf)
			return nil
		}
		return nil
	}
	ssb.MessageTypes["git-repo"] = func() interface{} {
		return &Repo{}
	}
	ssb.MessageTypes["git-update"] = func() interface{} {
		return &RepoUpdate{}
	}
}
