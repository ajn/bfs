package bfss3_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/bsm/bfs"
	"github.com/bsm/bfs/bfss3"
	"github.com/bsm/bfs/testdata/lint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const bucketName = "bsm-bfs-unittest"

var awsConfig = aws.Config{Region: aws.String("us-east-1")}

var _ = Describe("Bucket", func() {
	var data = lint.Data{}

	BeforeEach(func() {
		prefix := "x/" + strconv.FormatInt(time.Now().UnixNano(), 10)
		subject, err := bfss3.New(bucketName, &bfss3.Config{Prefix: prefix, AWS: awsConfig})
		Expect(err).NotTo(HaveOccurred())

		readonly, err := bfss3.New(bucketName, &bfss3.Config{Prefix: "m/", AWS: awsConfig})
		Expect(err).NotTo(HaveOccurred())

		data.Subject = subject
		data.Readonly = readonly
	})

	Context("defaults", lint.Lint(&data))
})

// ------------------------------------------------------------------------

func TestSuite(t *testing.T) {
	if err := sandboxCheck(); err != nil {
		t.Skipf("skipping test, no sandbox access: %v", err)
		return
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "bfs/bfss3")
}

func sandboxCheck() error {
	ctx := context.Background()
	b, err := bfss3.New(bucketName, &bfss3.Config{AWS: awsConfig})
	if err != nil {
		return err
	}
	defer b.Close()

	if _, err := b.Head(ctx, "____"); err != bfs.ErrNotFound {
		return err
	}
	return nil
}

var _ = AfterSuite(func() {
	ctx := context.Background()
	b, err := bfss3.New(bucketName, &bfss3.Config{Prefix: "x/", AWS: awsConfig})
	Expect(err).NotTo(HaveOccurred())
	defer b.Close()

	it, err := b.Glob(ctx, "**")
	Expect(err).NotTo(HaveOccurred())
	defer it.Close()

	for it.Next() {
		Expect(b.Remove(ctx, it.Name())).To(Succeed())
	}
	Expect(it.Error()).NotTo(HaveOccurred())
})
