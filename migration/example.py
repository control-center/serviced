
import servicemigration as sm
sm.require(sm.version.API_VERSION)

svc = sm.getServices({
	"Title": "zproxy"
})[0]

svc.setDescription("an_unlikely-description")

sm.commit()

