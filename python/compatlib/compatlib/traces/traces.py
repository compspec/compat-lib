import os

import pandas

import compatlib.utils as utils
from compatlib.models.distance import calculate_levenshtein


class Event:
    """
    An event object holds arbitrary event metadata
    """

    def __init__(self, **kwargs):
        for key, value in kwargs.items():
            setattr(self, key, value)


class TraceSet:
    """
    A Set of traces to operate on.

    Lookup: is typically only done once
    Open: is when the file is read
    """

    def __init__(self, files=None):
        self.files = files or []
        self.check()

    def count(self):
        return len(self.files)

    def check(self):
        """
        Ensure that all trace files provided actually exist.
        """
        events = []
        for filename in self.files:
            filename = os.path.abspath(filename)
            if not os.path.exists(filename):
                raise ValueError(f"{filename} does not exist")
            events.append(filename)
        self.files = events

    def iter_events(self, operation="Open"):
        """
        Iterate through files and yield event object
        """
        for filename in self.files:
            basename = os.path.basename(filename)
            for line in utils.read_file(filename).split("\n"):
                if not line:
                    continue
                # date, time, golang-file  timestamp function path
                # 2024/11/08 10:46:19 recorder.go:46: 1731062779714551943 Lookup     /etc
                parts = [x for x in line.split() if x]
                if parts[-2] != operation:
                    continue
                yield Event(
                    filename=filename,
                    basename=basename,
                    function=parts[-2],
                    path=parts[-1],
                    timestamp=int(parts[-3]),
                    normalized_path=utils.normalize_soname(parts[-1]),
                )

    def to_dataframe(self, operation="Open"):
        """
        Create a data frame of lookup values, we can save for later and derive paths from it.

        Normalized path removes the so version, if we find it. ms_in_state is milliseconds in state
        and is the time from the current event to the next, which is the time spent in that event.
        """
        df = pandas.DataFrame(
            columns=[
                "filename",
                "basename",
                "function",
                "path",
                "normalized_path",
                # This is a normalized path
                "previous_path",
                "timestamp",
                "ms_in_state",
            ]
        )
        idx = 0
        previous_timestamp = None
        previous_path = None
        current = None
        for event in self.iter_events(operation=operation):
            normalized_path = utils.normalize_soname(event.path)
            df.loc[idx, :] = [
                event.filename,
                os.path.basename(event.filename),
                event.function,
                event.path,
                normalized_path,
                previous_path,
                event.timestamp,
                None,
            ]
            if current is not None and event.filename != current:
                previous_path = None
                previous_timestamp = None
            if previous_timestamp is not None:
                df.loc[idx - 1, "ms_in_state"] = event.timestamp - previous_timestamp
            previous_timestamp = event.timestamp
            previous_path = normalized_path
            current = event.filename
            idx += 1
        return df

    def distance_matrix(self, operation="Open"):
        """
        Generate pairwise distance matrix for paths
        """
        lookup = self.as_paths(operation=operation)
        names = list(lookup.keys())
        df = pandas.DataFrame(index=names, columns=names)
        for filename1 in names:
            for filename2 in names:
                if filename1 > filename2:
                    continue
                aligned1, aligned2 = align_paths(lookup[filename1], lookup[filename2])
                distance = calculate_levenshtein(aligned1, aligned2)
                df.loc[filename1, filename2] = float(distance)
                df.loc[filename2, filename1] = float(distance)
        return df

    @property
    def samples(self):
        return list(self.as_paths().values())

    def iter_loo(self):
        """
        Yield train and test data for leave 1 out samples.
        Return the name of the test sample to identify it.
        I mostly just wanted to call this function "iter_loo" :)
        """
        paths = self.as_paths()
        for left_out in paths:
            train = [v for k, v in paths.items() if k != left_out]
            test = paths[left_out]
            yield train, test, left_out

    def as_paths(self, fullpath=False, operation="Open", remove_so_version=True):
        """
        Return lists of paths (lookup) corresponding to traces.
        """
        lookup = {}
        for event in self.iter_events(operation=operation):
            key = event.basename
            if fullpath:
                key = event.filename
            if key not in lookup:
                lookup[key] = []
            if remove_so_version:
                lookup[key].append(event.normalized_path)
            else:
                lookup[key].append(event.path)
        return lookup

    def all_counts(self, operation="Open", remove_so_version=True):
        """
        Return lookup of all counts corresponding to traces.

        Since we just have one lookup, this one is returned
        sorted.
        """
        lookup = {}
        for event in self.iter_events(operation=operation):
            path = event.path
            if remove_so_version:
                path = event.normalized_path
            if path not in lookup:
                lookup[path] = 0
            lookup[path] += 1
        return dict(sorted(lookup.items(), key=lambda item: item[1], reverse=True))

    def as_counts(self, fullpath=False, operation="Open", remove_so_version=True):
        """
        Return lookup of counts corresponding to traces.
        """
        lookup = {}
        for event in self.iter_events(operation=operation):
            key = event.basename
            if fullpath:
                key = event.filename
            if event.filename not in lookup:
                lookup[key] = {}
            path = event.path
            if remove_so_version:
                path = event.normalized_path
            if path not in lookup[key]:
                lookup[key][path] = 0
            lookup[key][path] += 1
        return lookup


# Alignment helpers


def align_paths(paths1, paths2, match_score=1, mismatch_score=-1, gap_penalty=-1):
    """
    This is a hacked Needleman-Wunsch algorithm for global sequence alignment,
    but instead of handling a sequence (string) we do two lists of paths
    """
    # Initialize the scoring matrix
    rows = len(paths1) + 1
    cols = len(paths2) + 1
    matrix = [[0 for _ in range(cols)] for _ in range(rows)]

    # Fill the first row and column with gap penalties
    for i in range(1, rows):
        matrix[i][0] = matrix[i - 1][0] + gap_penalty
    for j in range(1, cols):
        matrix[0][j] = matrix[0][j - 1] + gap_penalty

    # Calculate the scores for the rest of the matrix
    for i in range(1, rows):
        for j in range(1, cols):
            match = matrix[i - 1][j - 1] + (
                match_score if paths1[i - 1] == paths2[j - 1] else mismatch_score
            )
            delete = matrix[i - 1][j] + gap_penalty
            insert = matrix[i][j - 1] + gap_penalty
            matrix[i][j] = max(match, delete, insert)

    # Backtrack to find the optimal alignment
    alignment1 = []
    alignment2 = []
    i = rows - 1
    j = cols - 1
    while i > 0 or j > 0:
        if (
            i > 0
            and j > 0
            and matrix[i][j]
            == matrix[i - 1][j - 1]
            + (match_score if paths1[i - 1] == paths2[j - 1] else mismatch_score)
        ):
            alignment1 = [paths1[i - 1]] + alignment1
            alignment2 = [paths2[j - 1]] + alignment2
            i -= 1
            j -= 1
        elif i > 0 and matrix[i][j] == matrix[i - 1][j] + gap_penalty:
            alignment1 = [paths1[i - 1]] + alignment1
            alignment2 = [""] + alignment2
            i -= 1
        else:
            alignment1 = [""] + alignment1
            alignment2 = [paths2[j - 1]] + alignment2
            j -= 1

    return alignment1, alignment2
