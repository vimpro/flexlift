document.querySelectorAll(".thumbnail").forEach(element => {
    element.addEventListener("click", () => {
        element.classList.toggle("thumbnail-full")
    })
});

function like(element) {
    let uuid = element.parentElement.id
    let likeCounter = element.parentElement.querySelector(`#likeCounter`)
    let likeButton = element.parentElement.querySelector(`#like`)

    if (!window.signedIn) {
        likeButton.innerText = "Sign in to like"
        return
    }

    let count = Number(likeCounter.getAttribute('count'))

    if (likeButton.classList.contains('liked')) {
        fetch(`/removeLike/${uuid}`, {method: "POST"})
        count -= 1
        likeCounter.innerText = `${count} likes`
        likeButton.classList.remove('liked')
        likeButton.innerText = "Like post"
    } else {
        fetch(`/likePost/${uuid}`, {method: "POST"})
        count += 1
        likeCounter.innerText = `${count} likes`
        likeButton.classList.add('liked')
        likeButton.innerText = "Remove like"
    }
    likeCounter.setAttribute('count', count.toString())
}