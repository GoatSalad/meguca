import { Post, PostView } from '../posts'
import { fileTypes, PostLink, PostData, ThreadData } from "../common"
import { posts as postCollection, hidden, mine } from '../state'
import { pluralize, escape, threads } from '../util'
import options from "../options"
import lang from "../lang"
import { setSyncCounter } from "../connection"
import { notifyAboutReply, setTitle } from "../ui"
import { extractConfigs, isBanned } from "."

const counters = document.getElementById("thread-post-counters")
let postCtr = 0,
    imgCtr = 0

// Render the HTML of a thread page
export default function (html: string) {
    if (html) {
        threads.innerHTML = html
    }
    if (isBanned()) {
        return
    }
    extractConfigs()

    const data = JSON.parse(
        threads.querySelector("#post-data").textContent,
    ) as ThreadData
    const {posts} = data
    setSyncCounter(data.logCtr)
    delete data.posts

    setPostCount(data.postCtr, data.imageCtr)

    extractPost(data, data.op)
    postCollection.lowestID = posts.length ? posts[0].id : data.id
    if (data.image) {
        data.image.large = true
    }

    // Extra client-side localizations. Not done server-side for better
    // cacheability.
    localizeOmitted()
    if (options.anonymise) {
        for (let el of threads.querySelectorAll(".name")) {
            el.textContent = lang.posts["anon"]
        }
    } else if (options.lang !== "en_GB") { // Server renders in en_GB
        // Localize posts without a poster name or tripcode
        for (let el of threads.querySelectorAll(".name")) {
            if (el.textContent === "Anonymous") {
                el.textContent = lang.posts["anon"]
            }
        }

        // Localize banned post notices
        for (let el of threads.querySelectorAll(".banned")) {
            el.innerText = lang.posts["banned"]
        }
    }

    setThreadTitle(data)

    for (let post of posts) {
        extractPost(post, data.op)
    }

    // If the post is still open, rerender its body, to sync the parser state.
    // Needs to be done after models are populated to resolve temporary image
    // links in open posts.
    for (let m of postCollection) {
        if (m.editing) {
            m.view.reparseBody()
        }
    }
}

// Set thread title to tab
export function setThreadTitle(data: ThreadData) {
    setTitle(`/${data.board}/ - ${escape(data.subject)} (#${data.id})`)
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting
function extractPost(post: PostData, op: number) {
    const el = document.getElementById(`p${post.id}`)

    if (hidden.has(post.id)) {
        return el.remove()
    }
    post.op = op

    const model = new Post(post),
        view = new PostView(model, el)
    postCollection.add(model)

    // Apply client-specific formatting to a post rendered server-side

    // Render time-zone correction or relative time. Will do unneeded work,
    // if client is on UTC. Meh.
    view.renderTime()

    // Localize staff titles
    if (post.auth && options.lang != "en_GB") {
        view.renderName()
    }

    const {model: {links, backlinks, image}} = view
    localizeLinks(links, view, true)
    localizeLinks(backlinks, view, false)

    if (image) {
        const should = options.hideThumbs
            || options.workModeToggle
            || (image.spoiler && !options.spoilers)
            || (image.fileType === fileTypes.gif && options.autogif)
        if (should) {
            view.renderImage(false)
        }
    }
}

// Localize omitted post and image span
function localizeOmitted() {
    // Server renders in en_GB
    if (options.lang === "en_GB") {
        return
    }
    const el = document.querySelector(".omit")
    if (!el) {
        return
    }

    const posts = parseInt(el.getAttribute("data-omit")),
        images = parseInt(el.getAttribute("data-image-omit"))
    let text = pluralize(posts, lang.plurals["post"])
    if (images) {
        text += ` ${lang.posts["and"]} `
            + pluralize(images, lang.plurals["image"])
    }
    text += ` ${lang.posts["omitted"]} `

    el.firstChild.replaceWith(text)
    el.querySelector("a.history").textContent = lang.posts["seeAll"]
}

// Add (You) to posts linking to the user's posts and trigger desktop
// notifications, if needed
function localizeLinks(links: PostLink[], view: PostView, notify: boolean) {
    if (!links) {
        return
    }
    for (let [id] of links) {
        if (!mine.has(id)) {
            continue
        }
        for (let el of view.el.querySelectorAll(`a[data-id="${id}"]`)) {
            // Can create doubles with circular quotes. Avoid that.
            if (!el.textContent.includes(lang.posts["you"])) {
                el.textContent += " " + lang.posts["you"]
            }
        }
        if (notify) {
            notifyAboutReply(view.model)
        }
    }
}

// Increment thread post counters and rerender the indicator in the banner
export function incrementPostCount(post: boolean, hasImage: boolean) {
    if (post) {
        postCtr++
    }
    if (hasImage) {
        imgCtr++
    }
    renderPostCounter()
}

// Externally set thread image post count
export function setPostCount(posts: number, images: number) {
    postCtr = posts
    imgCtr = images
    renderPostCounter()
}

function renderPostCounter() {
    let text: string
    if (!postCtr && !imgCtr) {
        text = ""
    } else {
        text = `${postCtr} / ${imgCtr}`
    }
    counters.textContent = text
}
