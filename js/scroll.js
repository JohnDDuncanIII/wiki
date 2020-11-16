const toggle = document.getElementById("toggle");
if (toggle) {
	toggle.onclick = toggleToc;
}
function toggleToc(e) {
	const listDisplay = document.getElementById("toc_list").style.display;

	if (listDisplay != "none") {
		listDisplay = "none";
	} else {
		listDisplay = "block";
	}
}

const save = document.getElementById("bakecookie");
const names = document.getElementById("name");
const email = document.getElementById("email");
const xface = document.getElementById("xface");
const face = document.getElementById("face");
const homepage = document.getElementById("homepage");

if (localStorage.getItem('save')) {
	bakecookie.checked = true;
}

if (bakecookie.checked) {
	if (localStorage.getItem('name') != null) {
		names.value = localStorage.getItem('name');
	}
	if (localStorage.getItem('email') != null) {
		email.value = localStorage.getItem('email');
	}
	if (localStorage.getItem('xface') != null) {
		xface.value = localStorage.getItem('xface');
	}
	if (localStorage.getItem('face') != null) {
		face.value = localStorage.getItem('face');
	}
	if (localStorage.getItem('homepage') != null) {
		homepage.value = localStorage.getItem('homepage');
	}
}

bakecookie.onclick = function () {
	if (bakecookie.checked) {
		localStorage.setItem('save', true);
		localStorage.setItem('name', names.value);
		localStorage.setItem('email', email.value);
		localStorage.setItem('xface', xface.value);
		localStorage.setItem('face', face.value);
		localStorage.setItem('homepage', homepage.value);
	} else {
		localStorage.removeItem('save');
		localStorage.removeItem('name');
		localStorage.removeItem('email');
		localStorage.removeItem('xface');
		localStorage.removeItem('face');
		localStorage.removeItem('homepage');
	}
}

const img = document.getElementById("settings");
const overlay = document.getElementById("overlay");
const popupBox = document.getElementById("popup");

const jsScroll = document.getElementById("jsScroll");
const jsHeader = document.getElementById("jsHeader");

const comments = document.getElementById("comments");
const submit = document.getElementById("comment");
if (comments) {
	const jsComments = document.getElementById("jsComments");

	jsComments.checked = (localStorage.getItem('comments') == 'true');
	jsComments.onclick = function () {
		if (jsComments.checked) {
			localStorage.setItem('comments', true);
			comments.style.display = "block";
			submit.style.display = "block";
		} else {
			localStorage.setItem('comments', false);
			comments.style.display = "none";
			submit.style.display = "none";
		}
	}
	if (!jsComments.checked) {
		comments.style.display = "none";
		submit.style.display = "none";
	}
}

jsScroll.checked = (localStorage.getItem('scrollbar') == 'true');
jsHeader.checked = (localStorage.getItem('header') == 'true');

const body = document.body;

const header = document.getElementById("header");
const headerDisplay = header.style.display;

const c2 = document.getElementById("c2");
const c2Display = header.style.display;

var distToTop = 0;

if (!jsHeader.checked) {
	header.style.display = "none";
	c2.style.display = "none"
}
jsHeader.onclick = function () {
	if (jsHeader.checked) {
		localStorage.setItem('header', true);
		header.style.display = headerDisplay;
		c2.style.display = c2Display;
	}
	else {
		localStorage.setItem('header', false);
		header.style.display = "none";
		c2.style.display = "none";
	}
	calcDistToTop();
}
jsScroll.onclick = function () {
	if (jsScroll.checked) {
		const scrollEle = document.getElementById("scrollbar");

		if (scrollEle) {
			scrollEle.style.display = "block";
		} else {
			createScrollbar();
		}

		localStorage.setItem('scrollbar', true);
	} else {
		const scrollEle = document.getElementById("scrollbar");
		if (scrollEle) {
			scrollEle.style.display = "none";
		}
		localStorage.setItem('scrollbar', false);
	}
}

let isDimmed = false;
img.onclick = function() {
	if (isDimmed) {
		body.removeChild(overlay);
		body.removeChild(popupBox);
	}
	else {
		body.appendChild(overlay);
		overlay.style.display = "block"
		body.appendChild(popupBox);
		popupBox.style.display = "block"

	}
	isDimmed = !isDimmed;
}
overlay.onclick =  function() {
	if (isDimmed) {
		body.removeChild(overlay);
		body.removeChild(popupBox);
	}
	else {
		body.appendChild(overlay);
		body.appendChild(popupBox);
	}
	isDimmed = !isDimmed;
}

if (jsScroll.checked) {
	createScrollbar();
}

function createScrollbar () {
	const post = document.getElementById("entry");
	const postHeight = post.offsetHeight;
	const scrollEle = document.createElement("div");
	scrollEle.id="scrollbar";
	document.documentElement.insertBefore(scrollEle, body);

	calcDistToTop();

	body.onscroll = function() {
		const startNum = document.documentElement.scrollTop - distToTop;
		const numbers = [startNum, postHeight]
		const ratio = Math.max.apply(Math, numbers) / 100
		const l = numbers.length
		for (let i = 0; i < l; i++) { 
			numbers[i] = Math.round(numbers[i] / ratio); 
		}
		if ((window.innerHeight + window.scrollY) >= body.offsetHeight) {
			scrollEle.style.width="100%";
		} else if (numbers[0] > 0) {
			scrollEle.style.width=numbers[0]+"%";
		} else { scrollEle.style.width="0%"; }
	}
}

function calcDistToTop() {
	distToTop = 0;
	var postBegin = document.getElementById("entry");

	while (postBegin != document.documentElement) {
		if (postBegin.style.display != "none") {
			distToTop += postBegin.offsetTop;
		}
		postBegin = postBegin.parentNode;
	}

	// https://bugzilla.mozilla.org/show_bug.cgi?id=255754
	if (
		typeof InstallTrigger !== 'undefined' && 
		getComputedStyle(body, null).getPropertyValue('border-top-width')
	) {
		distToTop += parseInt(getComputedStyle(body, null).getPropertyValue('border-top-width'), 10);
	}
}
