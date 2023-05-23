import {Constants} from "../../constants";
import {hideElements} from "../../protyle/ui/hideElements";
import {setEditMode} from "../../protyle/util/setEditMode";
import {fetchPost} from "../../util/fetch";
import {zoomOut} from "../../menus/protyle";
import {processRender} from "../../protyle/util/processCode";
import {highlightRender} from "../../protyle/markdown/highlightRender";
import {blockRender} from "../../protyle/markdown/blockRender";
import {disabledForeverProtyle, disabledProtyle, enableProtyle} from "../../protyle/util/onGet";
import {setStorageVal} from "../../protyle/util/compatibility";
import {closePanel} from "./closePanel";
import {showMessage} from "../../dialog/message";
import {getCurrentEditor} from "../editor";

const forwardStack: IBackStack[] = [];

const focusStack = (backStack: IBackStack) => {
    const protyle = getCurrentEditor().protyle;
    window.siyuan.storage[Constants.LOCAL_DOCINFO] = {
        id: backStack.id,
        action: backStack.callback,
    };
    setStorageVal(Constants.LOCAL_DOCINFO, window.siyuan.storage[Constants.LOCAL_DOCINFO]);
    hideElements(["toolbar", "hint", "util"], protyle);
    if (protyle.contentElement.classList.contains("fn__none")) {
        setEditMode(protyle, "wysiwyg");
    }

    const startEndId = backStack.endId.split(Constants.ZWSP);
    if (startEndId[0] === protyle.wysiwyg.element.firstElementChild.getAttribute("data-node-id") &&
        startEndId[1] === protyle.wysiwyg.element.lastElementChild.getAttribute("data-node-id")) {
        protyle.contentElement.scrollTo({
            top: backStack.scrollTop,
            behavior: "smooth"
        });
        return;
    }

    if (backStack.id !== protyle.block.rootID) {
        fetchPost("/api/block/getDocInfo", {
            id: backStack.id,
        }, (response) => {
            (document.getElementById("toolbarName") as HTMLInputElement).value = response.data.name === "Untitled" ? "" : response.data.name;
            protyle.background.render(response.data.ial, protyle.block.rootID);
            protyle.wysiwyg.renderCustom(response.data.ial);
        });
    }

    if (backStack.zoomId) {
        if (backStack.zoomId !== protyle.block.id) {
            fetchPost("/api/block/checkBlockExist", {id: backStack.id}, existResponse => {
                if (existResponse.data) {
                    zoomOut(protyle, backStack.id, undefined, false, () => {
                        protyle.contentElement.scrollTop = backStack.scrollTop;
                    });
                }
            });
        } else {
            protyle.contentElement.scrollTop = backStack.scrollTop;
        }
        return;
    }

    fetchPost("/api/filetree/getDoc", {
        id: backStack.id,
        startID: startEndId[0],
        endID: startEndId[1],
    }, getResponse => {
        protyle.block.parentID = getResponse.data.parentID;
        protyle.block.parent2ID = getResponse.data.parent2ID;
        protyle.block.rootID = getResponse.data.rootID;
        protyle.block.showAll = false;
        protyle.block.mode = getResponse.data.mode;
        protyle.block.blockCount = getResponse.data.blockCount;
        protyle.block.id = getResponse.data.id;
        protyle.block.action = backStack.callback;
        protyle.wysiwyg.element.setAttribute("data-doc-type", getResponse.data.type);
        protyle.wysiwyg.element.innerHTML = getResponse.data.content;
        processRender(protyle.wysiwyg.element);
        highlightRender(protyle.wysiwyg.element);
        blockRender(protyle, protyle.wysiwyg.element, backStack.scrollTop);
        if (getResponse.data.isSyncing) {
            disabledForeverProtyle(protyle);
        } else {
            if (protyle.disabled) {
                disabledProtyle(protyle);
            } else {
                enableProtyle(protyle);
            }
        }
        protyle.contentElement.scrollTop = backStack.scrollTop;
        protyle.breadcrumb?.render(protyle);
    });
};

export const pushBack = () => {
    const protyle = getCurrentEditor().protyle;
    window.siyuan.backStack.push({
        id: protyle.block.showAll ? protyle.block.id : protyle.block.rootID,
        endId: protyle.wysiwyg.element.firstElementChild.getAttribute("data-node-id") + Constants.ZWSP + protyle.wysiwyg.element.lastElementChild.getAttribute("data-node-id"),
        scrollTop: protyle.contentElement.scrollTop,
        callback: protyle.block.action,
        zoomId: protyle.block.showAll ? protyle.block.id : undefined
    });
};

export const goForward = () => {
    if (window.siyuan.menus.menu.element.classList.contains("b3-menu--fullscreen") &&
        !window.siyuan.menus.menu.element.classList.contains("fn__none")) {
        window.siyuan.menus.menu.element.dispatchEvent(new CustomEvent("click", {detail: "back"}));
        return;
    } else if (document.getElementById("model").style.transform === "translateY(0px)" ||
        document.getElementById("menu").style.transform === "translateX(0px)" ||
        document.getElementById("sidebar").style.transform === "translateX(0px)") {
        closePanel();
        return;
    }
    if (window.JSAndroid && forwardStack.length < 2) {
        window.JSAndroid.returnDesktop();
        return;
    }
    if (forwardStack.length < 2) {
        return;
    }
    window.siyuan.backStack.push(forwardStack.pop());
    focusStack(forwardStack[forwardStack.length - 1]);
};

export const goBack = () => {
    const editor = getCurrentEditor();
    if (window.siyuan.menus.menu.element.classList.contains("b3-menu--fullscreen") &&
        !window.siyuan.menus.menu.element.classList.contains("fn__none")) {
        window.siyuan.menus.menu.element.dispatchEvent(new CustomEvent("click", {detail: "back"}));
        return;
    } else if (document.getElementById("model").style.transform === "translateY(0px)") {
        document.getElementById("model").style.transform = "";
        return;
    } else if (window.siyuan.viewer && !window.siyuan.viewer.destroyed) {
        window.siyuan.viewer.destroy();
        return;
    } else if (document.getElementById("menu").style.transform === "translateX(0px)" ||
        document.getElementById("sidebar").style.transform === "translateX(0px)") {
        closePanel();
        return;
    } else if (editor && !editor.protyle.toolbar.subElement.classList.contains("fn__none")) {
        hideElements(["util"], editor.protyle);
        closePanel();
        return;
    } else if (window.siyuan.dialogs.length !== 0) {
        hideElements(["dialog"]);
        closePanel();
        return;
    }
    if (window.JSAndroid && window.siyuan.backStack.length < 1) {
        if (document.querySelector('#message [data-id="exitTip"]')) {
            window.JSAndroid.returnDesktop();
        } else {
            showMessage(window.siyuan.languages.returnDesktop, 3000, "info", "exitTip");
        }
        return;
    }
    if (window.siyuan.backStack.length < 1) {
        return;
    }
    if (forwardStack.length === 0 && editor) {
        const protyle = editor.protyle;
        forwardStack.push({
            id: protyle.block.showAll ? protyle.block.id : protyle.block.rootID,
            endId: protyle.wysiwyg.element.firstElementChild.getAttribute("data-node-id") + Constants.ZWSP + protyle.wysiwyg.element.lastElementChild.getAttribute("data-node-id"),
            scrollTop: protyle.contentElement.scrollTop,
            callback: protyle.block.action,
            zoomId: protyle.block.showAll ? protyle.block.id : undefined
        });
    }
    const item = window.siyuan.backStack.pop();
    forwardStack.push(item);
    focusStack(item);
};
