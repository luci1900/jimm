const oldDomain = 'canonical-jaas-1.readthedocs-hosted.com';
const newDomain = 'canonical.com/juju/docs/jaas';

function overwriteURL(el) {
	if (!el) return;
	el.querySelectorAll('a').forEach(function (anchor) {
		const href = anchor.getAttribute('href');
		if (href && href.includes(oldDomain)) {
			anchor.href = href.replace(oldDomain, newDomain);
		}
	});
}

overwriteURL(document.querySelector('header'));
overwriteURL(document.querySelector('.rst-versions'));
