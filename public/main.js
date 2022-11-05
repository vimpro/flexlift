document.querySelectorAll(".thumbnail").forEach(element => {
    element.addEventListener("click", () => {
        element.classList.toggle("thumbnail-full")
    })
});