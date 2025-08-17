package gits

import "fmt"

func (r *Repo) Traverse(neg *Negotiation) (map[string]bool, error) {
	if neg == nil {
		neg = &Negotiation{}
	}

	result := map[string]bool{}
	visited := map[string]bool{}

	var walk func(hash string) error

	walk = func(hash string) error {
		if !neg.Haves[hash] {
			result[hash] = true
		}

		object, err := r.Object(hash)

		if err != nil {
			return err
		}

		if object.Type == OBJ_COMMIT {
			// Check if visited.
			if visited[hash] {
				return nil
			}

			// Traverse tree
			if object.TreeHash != "" {
				err := walk(object.TreeHash)

				if err != nil {
					return err
				}
			}

			// Traverse parents
			for _, parentHash := range object.ParentHashes {
				if parentHash != "" {
					err := walk(parentHash)

					if err != nil {
						return err
					}
				}
			}

			// Add to visited
			visited[hash] = true
		}

		if object.Type == OBJ_TREE {
			// Check if visited.
			if visited[hash] {
				return nil
			}

			// Traverse tree
			treeHashes, err := object.Tree()

			if err != nil {
				return err
			}

			for hash, typ := range treeHashes {
				if !neg.Haves[hash] {
					result[hash] = true
				}

				if typ == OBJ_TREE || typ == OBJ_COMMIT {
					err := walk(hash)

					if err != nil {
						return err
					}
				}
			}

			// Add to visited
			visited[hash] = true
		}

		if object.Type == OBJ_BLOB {
			// Nothing to do.
		}

		if object.Type == OBJ_TAG {
			return fmt.Errorf("tag object is not supported")
		}

		return nil
	}

	for want, include := range neg.Wants {
		if !include {
			continue
		}

		err := walk(want)

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
