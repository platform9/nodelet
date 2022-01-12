import requests


class Monocular(object):
    def __init__(self, client):
        self.client = client

    def get_repo(self, name):
        endpoint = 'v1/repos/{0}'.format(name)
        return self.client(endpoint)['data']

    def get_repos(self):
        endpoint = 'v1/repos'
        return self.client(endpoint)['data']

    def delete_repo(self, name):
        endpoint = 'v1/repos/{0}'.format(name)
        return self.client(endpoint, 'DELETE')

    def create_repo(self, repo):
        """
        :param repo dict: contains URL, name, and source attributes
        """
        endpoint = 'v1/repos'
        return self.client(endpoint, 'POST', repo)

    def get_or_create_repo(self, repo):
        try:
            return self.get_repo(repo['name'])
        except requests.HTTPError as e:
            if e.response.status_code != 404:
                raise
            return self.create_repo(repo)

    def get_charts(self):
        endpoint = 'v1/charts'
        return self.client(endpoint)['data']

    def get_release(self, name):
        endpoint = 'v1/releases/{0}'.format(name)
        return self.client(endpoint)['data']

    def get_releases(self):
        endpoint = 'v1/releases'
        return self.client(endpoint)['data']

    def create_release(self, release):
        endpoint = 'v1/releases'
        return self.client(endpoint, 'POST', release)

    def delete_release(self, name):
        endpoint = 'v1/releases/{0}'.format(name)
        return self.client(endpoint, 'DELETE')
