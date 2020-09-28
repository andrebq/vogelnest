package trail

type (
	fatalf interface {
		Fatalf(fmt string, args ...interface{})
	}
)

func mustNot(testlog fatalf, msg string, err error) {
	if err != nil {
		testlog.Fatalf("Must not %v: %v", msg, err)
	}
}
