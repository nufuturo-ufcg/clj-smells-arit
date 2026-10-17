(ns athens.views.left-sidebar.tasks
  (:require
    ["/components/Block/Taskbox"    :refer [Taskbox]]
    ["/components/Empty/Empty"      :refer [Empty
                                            EmptyIcon
                                            EmptyTitle
                                            EmptyMessage]]
    ["/components/Icons/Icons"      :refer [FilterCircleIcon FilterCircleFillIcon CheckboxIcon]]
    ["/components/Widget/Widget"    :refer [Widget WidgetHeader WidgetBody WidgetTitle WidgetToggle]]
    ["@chakra-ui/react"             :refer [FormControl Select FormLabel Heading Popover PopoverTrigger PopoverAnchor PopoverContent PopoverBody Portal IconButton Link Text VStack Flex Link Flex]]
    ["framer-motion"                :refer [motion AnimatePresence]]
    [athens.parse-renderer          :as parse-renderer]
    [athens.reactive                :as reactive]
    [athens.router                  :as router]
    [athens.types.query.shared      :as query]
    [athens.types.tasks.handlers    :as task-handlers]
    [athens.types.tasks.shared      :as shared]
    [athens.views.left-sidebar.subs :as left-sidebar-subs]
    [re-frame.core                  :as rf]
    [reagent.core                   :as r]))
