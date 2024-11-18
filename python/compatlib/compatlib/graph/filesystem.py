import os

import matplotlib.pylab as plt
import networkx as nx
import numpy
from matplotlib.colors import TwoSlopeNorm

import compatlib.utils as utils


class INode:
    """
    An INode is part of a Filesystem Trie
    We keep track of a count since we are going to use
    this to plot frequency of access.
    """

    def __init__(self, name, count=0):
        self.children = {}
        self.name = name
        self.count = count

    @property
    def basename(self):
        return os.path.basename(self.name) or os.sep

    @property
    def label(self):
        if self.count == 0:
            return self.basename
        return f"{self.basename}\n{self.count}"

    def increment(self, count):
        self.count += count


class Filesystem:
    """
    A Filesystem is a Trie of nodes

    # How to interact with fs
    # This path is not in the top 10, so will return None
    inode = fs.find("/opt/lammps/examples/reaxff/HNS/in.reaxff.hns")

    # This path IS in the top 10, so we get a node back
    inode = fs.find("/opt/lammps/examples/reaxff/HNS/ffield.reax.hns")
    print(f"{inode.name} is recorded {inode.count} times across recordings.")
    """

    def __init__(self):
        self.root = INode(os.sep)
        self.min_count = 0
        self.max_count = 0

    def get_graph(self, font_size=10, tree=True, node_size=1000, title=None):
        """
        Get a plot for a trie
        """
        plt.figure(figsize=(20, 8))
        graph = nx.DiGraph()

        # Recursive function to walk through root, etc.
        color_counts = {}
        get_counts(counts=color_counts, node=self.root)
        add_to_graph(graph=graph, root=self.root, node=self.root)

        # Set a filter for the highest color so it doesn't bias the entire plot
        unique_counts = list(set(list(color_counts.values())))
        unique_counts.sort()
        without_outliers = reject_outliers(unique_counts)

        # Color based on count (outliers removed)!
        min_count = without_outliers[0]
        max_count = without_outliers[-1]
        colors = derive_node_colors(min_count=min_count, max_count=max_count + 1)

        # We only want to get colors that match the count, so the scale is relevant
        node_colors = []

        # Also update node labels to show count
        new_labels = {}
        for i, node in enumerate(graph.nodes):
            if node == "/":
                count = 0
                new_labels[node] = node
            else:
                count = color_counts[node]
                # Don't put label if count is 0!
                if count == 0:
                    new_labels[node] = node
                else:
                    new_labels[node] = f"{node}\n{count}"

            # This adjusts the color scale within the range
            # that does not have outliers
            if count < min_count:
                count = min_count
            if count > max_count:
                count = max_count
            node_colors.append(colors[count])

        # Tree visualization (much better) requires graphviz, dot, etc.
        if tree:
            for i, layer in enumerate(nx.topological_generations(graph)):
                for n in layer:
                    graph.nodes[n]["layer"] = i
            pos = nx.multipartite_layout(graph, subset_key="layer", align="horizontal")

            # Flip the layout so the root node is on top
            for k in pos:
                pos[k][-1] *= -1
        else:
            pos = nx.spring_layout(graph)

        # This will plot, and the user can call plt.show() or plt.savefig()
        title = title or "Filesystem Recording Trie"
        plt.title(title)
        nx.draw(
            graph,
            pos,
            with_labels=False,
            alpha=0.5,
            node_size=node_size,
            node_color=node_colors,
            font_size=font_size,
        )

        # Update and rotate the labels a bit
        for node, (x, y) in pos.items():
            plt.text(x, y, new_labels[str(node)], rotation=45, ha="center", va="center")
        plt.tight_layout()
        return graph

    def insert(self, path, count=0, remove_so_version=True):
        """
        Insert an INode into the filesystem.

        If we are adding a count, increment by it. We also build the tree
        without .so.<version> to compare across.
        """
        if remove_so_version:
            path = utils.normalize_soname(path)
        node = self.root
        partial_path = []
        for part in path.split(os.sep):
            partial_path.append(part)
            if part not in node.children:
                assembled_path = os.sep.join(partial_path)
                node.children[part] = INode(assembled_path)
            node = node.children[part]
        node.increment(count)

        # Update counts
        if node.count < self.min_count:
            self.min_count = node.count
        if node.count > self.max_count:
            self.max_count = node.count

    def find(self, path):
        """
        Search the filesystem for a path
        """
        node = self.root
        for part in path.split(os.sep):
            if part not in node.children:
                return
            node = node.children[part]
        return node


def reject_outliers(data, m=2.0):
    """
    Reject outliers to derive the color scale.
    """
    # Median isn't influenced by outliers
    d = numpy.abs(data - numpy.median(data))
    mdev = numpy.median(d)
    s = d / mdev if mdev else numpy.zeros(len(d))
    return numpy.array(data)[s < m]


# Plotting helpers


def add_to_graph(graph, root, node, parent=None):
    """
    Helper function to add node to graph
    """
    if node != root:
        # This is probably a bug, just skip for now
        if node.basename != parent.basename:
            graph.add_edge(parent.basename, node.basename)
    for path, child in node.children.items():
        add_to_graph(graph=graph, root=root, node=child, parent=node)


def derive_node_colors(min_count, max_count):
    """
    Given the min, max, and a center, return a range of colors
    """
    palette = plt.cm.get_cmap("viridis")
    center = int(abs(max_count - min_count) / 2)
    norm = TwoSlopeNorm(vmin=min_count, vcenter=center, vmax=max_count)
    return [palette(norm(c)) for c in range(min_count, max_count)]


def get_counts(counts={}, node=None):
    """
    Get a flat list of counts
    """
    for path, child in node.children.items():
        counts[path] = child.count
        get_counts(counts, child)
