package builds

import (
	"fmt"
	"path/filepath"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	kapi "k8s.io/kubernetes/pkg/api"
)

var _ = g.Describe("[builds][Slow] can use build secrets", func() {
	defer g.GinkgoRecover()
	var (
		buildSecretBaseDir    = exutil.FixturePath("testdata", "build-secrets")
		secretsFixture        = filepath.Join(buildSecretBaseDir, "test-secret.json")
		secondSecretsFixture  = filepath.Join(buildSecretBaseDir, "test-secret-2.json")
		isFixture             = filepath.Join(buildSecretBaseDir, "test-is.json")
		dockerBuildFixture    = filepath.Join(buildSecretBaseDir, "test-docker-build.json")
		dockerBuildDockerfile = filepath.Join(buildSecretBaseDir, "Dockerfile")
		sourceBuildFixture    = filepath.Join(buildSecretBaseDir, "test-s2i-build.json")
		sourceBuildBinDir     = filepath.Join(buildSecretBaseDir, "s2i-binary-dir")
		oc                    = exutil.NewCLI("build-secrets", exutil.KubeConfigPath())
	)

	g.Describe("build with secrets", func() {
		oc.SetOutputDir(exutil.TestContext.OutputDir)

		g.It("should contain secrets during the source strategy build", func() {
			g.By("creating secret fixtures")
			err := oc.Run("create").Args("-f", secretsFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			err = oc.Run("create").Args("-f", secondSecretsFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating test image stream")
			err = oc.Run("create").Args("-f", isFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating test build config")
			err = oc.Run("create").Args("-f", sourceBuildFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("starting the test source build")
			out, err := oc.Run("start-build").Args("test", "--from-dir", sourceBuildBinDir).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "test-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("test", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("getting the image name")
			image, err := exutil.GetDockerImageReference(oc.REST().ImageStreams(oc.Namespace()), "test", "latest")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("verifying the build secrets were available during build and not present in the output image")
			pod := exutil.GetPodForContainer(kapi.Container{Name: "test", Image: image})
			oc.KubeFramework().TestContainerOutput("test-build-secret-source", pod, 0, []string{
				"testsecret/secret1=secret1",
				"testsecret/secret2=secret2",
				"testsecret/secret3=secret3",
				"testsecret2/secret1=secret1",
				"testsecret2/secret2=secret2",
				"testsecret2/secret3=secret3",
			})
		})

		g.It("should contain secrets during the docker strategy build", func() {
			g.By("creating secret fixtures")
			err := oc.Run("create").Args("-f", secretsFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			err = oc.Run("create").Args("-f", secondSecretsFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating test image stream")
			err = oc.Run("create").Args("-f", isFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating test build config")
			err = oc.Run("create").Args("-f", dockerBuildFixture).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("starting the test docker build")
			out, err := oc.Run("start-build").Args("test", "--from-file", dockerBuildDockerfile).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "test-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("test", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("getting the image name")
			image, err := exutil.GetDockerImageReference(oc.REST().ImageStreams(oc.Namespace()), "test", "latest")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("verifying the secrets are present in container output")
			pod := exutil.GetPodForContainer(kapi.Container{Name: "test", Image: image})
			oc.KubeFramework().TestContainerOutput("test-build-secret-docker", pod, 0, []string{
				"secret1=secret1",
				"relative-secret2=secret2",
			})

		})

	})
})
